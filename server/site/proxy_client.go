package site

import (
	"net"
)

type ProxyClient struct {
	Id        string
	Conn      *net.TCPConn

	InChan    chan ([]byte)
	OutChan   chan ([]byte)

	ChannelId uint8

	IsClosed  bool
}

func (this *ProxyClient) Close() {
	this.IsClosed = true
	this.Conn.Close()
}

func NewProxyClient(conn *net.TCPConn) *ProxyClient {
	return &ProxyClient{
		Id:conn.RemoteAddr().String(),
		Conn:conn,
		InChan:make(chan ([]byte), 100),
		OutChan:make(chan ([]byte), 100),
		IsClosed:false,
	}
}