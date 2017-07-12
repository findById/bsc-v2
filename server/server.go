package main

import (
	"bsc-v2/server/client"
	"net"
	"log"
	"bsc-v2/server/site"
	"bsc-v2/server/handler"
)

type ProxyServer struct {
	cm       *client.ClientManager
	pcm      *site.ProxyClientManager
	listener *net.TCPListener
	limiter  string
}

func NewProxyServer() *ProxyServer {
	cm := client.NewClientManager()
	pcm := site.NewProxyClientManager()
	return &ProxyServer{
		cm: cm,
		pcm:pcm,
	}
}

func (this *ProxyServer) Start(dataPort, userPort string) {
	// 开启与客户端数据传输端口
	err := this.listenDataPort(dataPort)
	if err != nil {
		log.Panic(err)
		return
	}
	// 开启用户访问端口
	err = this.listenUserPort(userPort)
	if err != nil {
		log.Panic(err)
		return
	}
}

/**
接收客户端发起的连接
 */
func (this *ProxyServer) listenDataPort(addr string) (err error) {
	laddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		log.Panic(err)
		return
	}
	listener, err := net.ListenTCP("tcp", laddr)
	if err != nil {
		log.Panic(err)
		return
	}
	go func() {
		for {
			conn, err := listener.AcceptTCP()
			if err != nil {
				continue
			}
			go this.handleDataConnection(conn)
		}
	}()
	return
}

/**
处理客户端发起的连接 (数据传输)
 */
func (this *ProxyServer) handleDataConnection(conn *net.TCPConn) {
	h := handler.NewHandler(conn, this.cm, this.pcm)
	h.Start()
}

/**
接收用户端发起的请求连接
 */
func (this *ProxyServer) listenUserPort(addr string) (err error) {
	laddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return
	}
	listener, err := net.ListenTCP("tcp", laddr)
	if err != nil {
		return
	}
	go func() {
		for {
			conn, err := listener.AcceptTCP()
			if err != nil {
				continue
			}
			go this.handleUserConnection(conn)
		}
	}()
	return
}

/**
处理用户端发起的请求
 */
func (this *ProxyServer) handleUserConnection(conn *net.TCPConn) {
	h := site.NewSiteHandler(conn, this.pcm, this.cm)
	h.Start()
}
