package http

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"strings"

	"github.com/chh-yu/goproxy/common"
)

type HttpServer struct {
	common.ServerBase
}

func (s *HttpServer) Run() error {
	address := fmt.Sprintf("%s:%d", s.IP, s.Port)
	listener, err := net.Listen("tcp", address)

	if err != nil {
		log.Panic(err)
	}
	// 死循环，每当遇到连接时，调用 handle
	for {
		client, err := listener.Accept()
		if err != nil {
			log.Panic(err)
		}
		go handle(client)
	}
}

func handle(client net.Conn) {
	if client == nil {
		return
	}
	defer client.Close()
	log.Printf("remote addr: %v\n", client.RemoteAddr())

	//从客户端获取数据
	var b [1024]byte
	n, err := client.Read(b[:])
	if err != nil {
		log.Println(err)
		return
	}

	var method, URL, address string
	// 从客户端数据读入 method，url
	fmt.Sscanf(string(b[:bytes.IndexByte(b[:], '\n')]), "%s%s", &method, &URL)
	hostPortURL, err := url.Parse(URL)
	if err != nil {
		log.Println(err)
		return
	}

	// 如果方法是 CONNECT，则为 https 协议
	if method == "CONNECT" {
		address = hostPortURL.Scheme + ":" + hostPortURL.Opaque
	} else { //否则为 http 协议
		address = hostPortURL.Host
		// 如果 host 不带端口，则默认为 80
		if strings.Index(hostPortURL.Host, ":") == -1 { //host 不带端口， 默认 80
			address = hostPortURL.Host + ":80"
		}
	}
	log.Printf("address: %s", address)

	//获得了请求的 host 和 port，向服务端发起 tcp 连接
	server, err := net.Dial("tcp", address)
	if err != nil {
		log.Println(err)
		return
	}
	//如果使用 https 协议，需先向客户端表示连接建立完毕
	if method == "CONNECT" {
		fmt.Fprint(client, "HTTP/1.1 200 Connection established\r\n\r\n")
	} else { //如果使用 http 协议，需将从客户端得到的 http 请求转发给服务端
		server.Write(b[:n])
	}
	//将客户端的请求转发至服务端，将服务端的响应转发给客户端。io.Copy 为阻塞函数，文件描述符不关闭就不停止
	go io.Copy(server, client)
	io.Copy(client, server)
}
