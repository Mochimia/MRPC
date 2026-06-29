package main

import (
	"MRPC/codec"
	"MRPC/server"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"
)

func startServer(addr chan string) {
	//启动tcp监听器
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatal("network error:", err)
	}
	log.Println("start rpc server on", l.Addr())
	//把地址传给main
	addr <- l.Addr().String()
	//接受客户端连接
	server.Accept(l)
}

func main() {
	addr := make(chan string)
	go startServer(addr)

	//简易的client
	//客户端从channel接收并连接服务端的地址
	conn, err := net.Dial("tcp", <-addr)
	if err != nil {
		log.Fatal("dial error:", err)
	}
	defer func() { _ = conn.Close() }()

	time.Sleep(time.Second)
	//发送 options
	_ = json.NewEncoder(conn).Encode(server.DefaultOption)
	cc := codec.NewGobCodec(conn)
	//发送 request&receive response
	for i := 0; i < 5; i++ {
		h := &codec.Header{
			ServiceMethod: "Foo.Sum",
			Seq:           uint64(i),
		}
		_ = cc.Write(h, fmt.Sprintf("MRPC req %d", h.Seq))
		_ = cc.ReadHeader(h)
		var reply string
		_ = cc.ReadBody(&reply)
		log.Println("reply:", reply)
	}
}
