package site

import (
	"github.com/findById/bsc-v2/core"
	"net"
)

const (
	TYPE_CLOSE = 1
)

type TcpClient struct {
	Id   string       // 用户端连接Id
	Conn *net.TCPConn // 用户端连接

	ClientId string // 对应用户端的客户端连接Id

	InChan    chan (core.Frame)
	OutChan   chan (core.Frame)
	CloseChan chan (int)

	ChannelId uint32 // 用户复用客户端连接的通道Id

	IsClosed bool
}

func (this *TcpClient) Close() {
	this.IsClosed = true
	this.Conn.Close()
	this.CloseChan <- TYPE_CLOSE
}

func NewTcpClient(conn *net.TCPConn) *TcpClient {
	return &TcpClient{
		Id:        conn.RemoteAddr().String(),
		Conn:      conn,
		InChan:    make(chan (core.Frame), 10000),
		OutChan:   make(chan (core.Frame), 10000),
		CloseChan: make(chan (int), 100),
		IsClosed:  false,
	}
}
