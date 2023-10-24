package main

import (
	"SimpleGoCache/cache"
	http2 "SimpleGoCache/http"
	"flag"
	"fmt"
	"log"
	"net/http"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func createGroup() *cache.Group {
	return cache.NewGroup("scores", 2<<10, cache.GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))
}

func startCacheServer(addr string, addrs []string, gee *cache.Group) {
	// 创建一个cache服务
	peers := http2.NewHTTPPool(addr)
	// 将peers添加进来，方便后面转发
	peers.Set(addrs...)
	// 将当前的cache服务注册到api服务中，用户后续的转发
	gee.RegisterPeers(peers)
	log.Println("geecache is running at", addr)
	// 启动cache服务
	log.Fatal(http.ListenAndServe(addr[7:], peers))
}

func startAPIServer(apiAddr string, gee *cache.Group) {
	// 添加api服务的接口
	http.Handle("/api", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			key := r.URL.Query().Get("key")
			view, err := gee.Get(key)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(view.ByteSlice())

		}))
	log.Println("fontend server is running at", apiAddr)
	// 启动服务
	log.Fatal(http.ListenAndServe(apiAddr[7:], nil))
}

func main() {
	var port int
	var api bool
	flag.IntVar(&port, "port", 8001, "Geecache server port")
	flag.BoolVar(&api, "api", false, "Start a api server?")
	flag.Parse()

	apiAddr := "http://localhost:9999"
	addrMap := map[int]string{
		8001: "http://localhost:8001",
		8002: "http://localhost:8002",
		8003: "http://localhost:8003",
	}

	var addrs []string
	for _, v := range addrMap {
		addrs = append(addrs, v)
	}

	gee := createGroup()
	if api {
		// 用来启动一个 API 服务（端口 9999），与用户进行交互
		go startAPIServer(apiAddr, gee)
	}
	// 用来启动缓存服务器，多次启动
	// ./server -port=8001
	// ./server -port=8002
	// ./server -port=8003
	startCacheServer(addrMap[port], addrs, gee)
}
