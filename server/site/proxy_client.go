package site

import (
	"net"
)

type ProxyClient struct {
	Id        string       // 用户端连接Id
	Conn      *net.TCPConn // 用户端连接

	ClientId  string       // 对应用户端的客户端连接Id

	InChan    chan ([]byte)
	OutChan   chan ([]byte)

	ChannelId uint8        // 用户复用客户端连接的通道Id

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
		InChan:make(chan ([]byte), 10000),
		OutChan:make(chan ([]byte), 10000),
		IsClosed:false,
	}
}