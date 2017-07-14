package main

import (
	"bsc-v2/server/client"
	"net"
	"log"
	"bsc-v2/server/handler"
	"bsc-v2/server/site"
	"time"
	"bsc-v2/core"
	"runtime"
)

const CHANNEL_SIZE int = 100

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
	ticker := time.NewTicker(time.Second * 10)
	go func() {
		for range ticker.C {
			log.Println("monitor CPU:", runtime.NumCPU(), "Routine:", runtime.NumGoroutine(), "Client:", this.cm.Size(), "ProxyClient:", this.pcm.Size())
			runtime.GC()
		}
	}()
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
	log.Println("handle data conn:", conn.RemoteAddr().String(), "Goroutine:", runtime.NumGoroutine())
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
	log.Println("handle proxy conn:", conn.RemoteAddr().String(), "Goroutine:", runtime.NumGoroutine())

	pc := site.NewProxyClient(conn)
	var c *client.Client

	// 查找可用连接通道
	client, success := this.searchConn(pc)
	if !success {
		// 没有找到空闲连接，发起通知客户端打开新连接
		if client == nil || client.IsClosed {
			log.Println("not found connect")
			pc.Close()
			return
		}
		// 告诉客户端打开新的连接接收数据
		data := core.NewFrame(core.NEW_CO, 0, core.NO_PAYLOAD)
		client.OutChan <- data
		log.Println("new connect")

		// 等待客户端打开新连接，10秒超时
		beginTime := time.Now().Unix()
		work: // 等待客户端发起可用连接，10秒超时
		for now := int64(0); (now - beginTime) < 10; now = time.Now().Unix() {
			// 等待客户端连接
			//for _, conn := range this.cm.CloneMap() {
			//	if conn == nil || conn.IsClosed {
			//		continue
			//	}
			//	if conn.ChannelIdSize() < CHANNEL_SIZE {
			//		pc.ChannelId = conn.NewChannelId()
			//		pc.ClientId = conn.Id
			//		c = conn // 复用当前可用数据通道
			//		this.pcm.Add(pc)
			//		//log.Println("new channel id", pc.ChannelId)
			//		break finded
			//	}
			//}
			client, success := this.searchConn(pc)
			if success {
				c = client
				break work
			}
		}
	} else {
		c = client
	}

	if c == nil {
		log.Println("not found connect")
		pc.Close()
		return
	}

	// 如该当前与客户端的连接大于1个，寻找空闲的连接并通知客户端关闭 (设计重复,待优化 proxy_transport_handler.go)
	if len(this.cm.CloneMap()) > 1 {
		for _, v := range this.cm.CloneMap() {
			if v != c && c.ChannelIdSize() < 1 {
				data := core.NewFrame(core.CLOSE_CO, pc.ChannelId, core.NO_PAYLOAD)
				v.OutChan <- data
			}
		}
	}

	log.Println("work cId:", c.Id, "pcId:", pc.Id, "channelId:", pc.ChannelId)
	h := site.NewSiteHandler(c, this.cm, pc, this.pcm)
	h.Start()
}

func (this *ProxyServer) searchConn(pc *site.ProxyClient) (*client.Client, bool) {
	var c *client.Client
	for _, conn := range this.cm.CloneMap() {
		if conn == nil || conn.IsClosed {
			continue
		}
		c = conn // 复用当前可用连接
		if conn.ChannelIdSize() < CHANNEL_SIZE {
			pc.ChannelId = conn.NewChannelId()
			pc.ClientId = conn.Id
			this.pcm.Add(pc)
			//log.Println("new channel id", pc.ChannelId)
			return c, true
		}
	}
	return c, false
}
