package main

import (
	"github.com/findById/bsc-v2/core"
	"github.com/findById/bsc-v2/server/client"
	"github.com/findById/bsc-v2/server/handler"
	"github.com/findById/bsc-v2/server/site"
	"log"
	"net"
	"runtime"
	"time"
)

const CHANNEL_SIZE int = 100

type ProxyServer struct {
	cm       *client.ProxyClientManager
	tcm      *site.ClientManager
	listener *net.TCPListener
	debug    bool
}

func NewProxyServer(token string, debug bool) *ProxyServer {
	cm := client.NewClientManager()
	tcm := site.NewClientManager()

	cm.AuthToken = token

	return &ProxyServer{
		cm:    cm,
		tcm:   tcm,
		debug: debug,
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
	ticker := time.NewTicker(time.Minute)
	go func() {
		for range ticker.C {
			log.Printf("monitor >> CPU:%d, Goroutine:%d, Client:%d, ProxyClient:%d\n", runtime.NumCPU(), runtime.NumGoroutine(), this.cm.Size(), this.tcm.Size())
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
	h := handler.NewProxyHandler(conn, this.cm, this.tcm, this.debug)
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
	log.Println("handle user conn:", conn.RemoteAddr().String(), "Goroutine:", runtime.NumGoroutine())

	tc := site.NewTcpClient(conn)
	var pc *client.ProxyClient

	// 查找可用连接通道
	client, success := this.searchConn(tc)
	if !success {
		// 没有找到空闲连接，发起通知客户端打开新连接
		// 没有找到空闲连接或连接已关闭, 代理客户端无法提供服务
		if client == nil || client.IsClosed {
			if this.debug {
				log.Println("not found connect")
			}
			tc.Close()
			return
		}
		// 告诉客户端打开新的连接接收数据
		data := core.NewFrame(core.NEW_CO, 0, core.NO_PAYLOAD)
		client.OutChan <- data
		if this.debug {
			log.Println("new connect")
		}

		// 等待客户端打开新连接，10秒超时
		beginTime := time.Now().Unix()
	work: // 等待客户端发起可用连接，10秒超时
		for now := int64(0); (now - beginTime) < 10; now = time.Now().Unix() {
			// 等待客户端连接
			client, success := this.searchConn(tc)
			if success {
				pc = client
				break work
			}
		}
	} else {
		pc = client
	}

	if pc == nil {
		if this.debug {
			log.Println("not found connect")
		}
		tc.Close()
		return
	}

	// 如该当前与客户端的连接大于1个，寻找空闲的连接并通知客户端关闭 (设计重复,待优化 proxy_transport_handler.go)
	if len(this.cm.CloneMap()) > 1 {
		for _, v := range this.cm.CloneMap() {
			if v != pc && pc.ChannelIdSize() < 1 {
				data := core.NewFrame(core.CLOSE_CO, tc.ChannelId, core.NO_PAYLOAD)
				v.OutChan <- data
			}
		}
	}

	log.Printf("user conn working >> pcId:%s, tcId:%s, chId:%v\n", pc.Id, tc.Id, tc.ChannelId)
	h := site.NewSiteHandler(pc, this.cm, tc, this.tcm, this.debug)
	h.Start()
}

/**
1. 如果没有代理客户端连接，返回 nil, false
2. 如果有代理客户端连接并且连接压力不大， 返回 conn, true
3. 如果有代理客户端连接，但是没有空闲连接，返回 conn, false
*/
func (this *ProxyServer) searchConn(tc *site.TcpClient) (*client.ProxyClient, bool) {
	var pc *client.ProxyClient
	for _, conn := range this.cm.CloneMap() {
		if conn == nil || conn.IsClosed {
			continue
		}
		pc = conn // 复用当前可用连接
		if conn.ChannelIdSize() < CHANNEL_SIZE {
			tc.ChannelId = conn.NewChannelId(tc.Id)
			tc.ClientId = conn.Id
			this.tcm.Add(tc)
			//log.Println("new channel id", pc.ChannelId)
			return pc, true
		}
	}
	return pc, false
}
