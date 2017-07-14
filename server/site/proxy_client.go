package site

import (
	"net"
	"bsc-v2/core"
)

const (
	TYPE_CLOSE = 1
)

type ProxyClient struct {
	Id        string       // 用户端连接Id
	Conn      *net.TCPConn // 用户端连接

	ClientId  string       // 对应用户端的客户端连接Id

	InChan    chan (core.Frame)
	OutChan   chan (core.Frame)
	CloseChan chan (int)

	ChannelId uint8        // 用户复用客户端连接的通道Id

	IsClosed  bool
}

func (this *ProxyClient) Close() {
	this.IsClosed = true
	this.Conn.Close()
	this.CloseChan <- TYPE_CLOSE
}

func NewProxyClient(conn *net.TCPConn) *ProxyClient {
	return &ProxyClient{
		Id:conn.RemoteAddr().String(),
		Conn:conn,
		InChan:make(chan (core.Frame), 10000),
		OutChan:make(chan (core.Frame), 10000),
		CloseChan:make(chan (int), 100),
		IsClosed:false,
	}
}