package client

import (
	"net"
	"bsc-v2/core"
	"sort"
)

type Client struct {
	Id         string // 客户端Id
	Conn       *net.TCPConn

	InChan     chan (core.Frame)
	OutChan    chan (core.Frame)

	channelIds []uint8

	IsClosed   bool
	IsAuthed   bool
}

func (this *Client) Close() {
	this.IsClosed = true
	this.Conn.Close()
}

func NewClient(conn *net.TCPConn) *Client {
	c := &Client{
		Id:         conn.RemoteAddr().String(),
		Conn:conn,
		InChan:make(chan (core.Frame), 100),
		OutChan:make(chan (core.Frame), 100),
		channelIds:make([]uint8, 0),
		IsClosed:   false,
		IsAuthed:   false,
	}
	return c
}

// channel

func (this *Client) NewChannelId() uint8 {
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

func (this *Client) RemoveChannelId(id uint8) {
	for i, cId := range this.channelIds {
		if cId == id {
			this.channelIds = append(this.channelIds[:i], this.channelIds[i + 1:]...)
			break
		}
	}
}

func (this *Client) ConsistsChannelId(id uint8) bool {
	for _, i := range this.channelIds {
		if i == id {
			return true
		}
	}
	return false
}

func (this *Client) ChannelIdSize() int {
	return len(this.channelIds)
}
