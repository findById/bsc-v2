package client

import (
	"net"
	"bsc-v2/core"
	"sort"
)

type ConnBean struct {
	Id         string
	Conn       *net.TCPConn

	InChan     chan (core.Frame)
	OutChan    chan (core.Frame)

	IsClosed   bool

	channelIds []uint8
}

func NewConnBean(conn *net.TCPConn) *ConnBean {
	bean := &ConnBean{
		Id:conn.RemoteAddr().String(),
		Conn:conn,
		InChan:make(chan (core.Frame), 100),
		OutChan:make(chan (core.Frame), 100),
		IsClosed:false,
		channelIds:make([]uint8, 0),
	}
	return bean
}

func (this *ConnBean) NewChannelId() uint8 {
	sort.Slice(this.channelIds, func(i, j int) bool {
		return this.channelIds[i] - this.channelIds[j] < 0
	})
	for i, id := range this.channelIds {
		if i != int(id) {
			this.channelIds = append(this.channelIds, id - 1)
			return id - 1
		}
	}
	this.channelIds = append(this.channelIds, 0)
	return 0
}

func (this *ConnBean) RemoveChannelId(id uint8) {
	for i, cId := range this.channelIds {
		if cId == id {
			this.channelIds = append(this.channelIds[:i], this.channelIds[i + 1:]...)
			break
		}
	}
}

func (this *ConnBean) ConsistsChannelId(id uint8) bool {
	for i := range this.channelIds {
		if i == id {
			return true
		}
	}
	return false
}

func (this *ConnBean) ChannelIdSize() int {
	return len(this.channelIds)
}

func (this *ConnBean) Close() {
	this.IsClosed = true
	this.Conn.Close()
}

type Client struct {
	Id       string               // 客户端Id

	Conn     map[string]*ConnBean // 数据传输连接

	IsClosed bool
	IsAuthed bool
}

func (this *Client) Close() {
	this.IsClosed = true
	for _, item := range this.Conn {
		item.Close()
	}
}

func NewClient(conn *net.TCPConn) *Client {
	c := &Client{
		Id:         conn.RemoteAddr().String(),
		Conn:       make(map[string]*ConnBean, 10),
		IsClosed:   false,
		IsAuthed:   false,
	}
	c.AddConn(conn)
	return c
}

func (this *Client) AddConn(conn *net.TCPConn) {
	if temp, ok := this.Conn[conn.RemoteAddr().String()]; ok {
		temp.Close()
	}
	this.Conn[conn.RemoteAddr().String()] = NewConnBean(conn)
}