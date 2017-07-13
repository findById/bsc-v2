package main

import (
	"bsc-v2/server/client"
	"net"
	"log"
	"bsc-v2/server/handler"
	"bsc-v2/server/site"
	"time"
	"bsc-v2/core"
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
	log.Println("handle data conn", conn.RemoteAddr().String())
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
	log.Println("handle user conn", conn.RemoteAddr().String())

	pc := site.NewProxyClient(conn)
	var c *client.Client

	beginTime := time.Now().Unix()
	a: // 未解决无客户端情况下接收到的连接
	for now := int64(0); (now - beginTime) < 10; now = time.Now().Unix() {
		// 查找可用连接通道 (临时解决方案)
		for _, conn := range this.cm.CloneMap() {
			if conn == nil || conn.IsClosed {
				continue
			}
			if conn.ChannelIdSize() < 10 {
				pc.ChannelId = conn.NewChannelId()
				pc.ClientId = conn.Id
				c = conn // 复用当前可用数据通道
				this.pcm.Add(pc)
				log.Println("new channel id", pc.ChannelId)
				break a
			} else {
				// 告诉客户端打开新的连接接收数据
				data := core.NewFrame(core.NEW_CO, 0, core.NO_PAYLOAD)
				conn.OutChan <- data

				log.Println("new connect")
				// 等待客户端连接
				time.Sleep(5 * time.Second)
			}
		}
	}

	h := site.NewSiteHandler(c, pc)
	h.Start()
}
