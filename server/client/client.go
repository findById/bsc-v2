package client

import (
	"net"
	"bsc-v2/core"
	"sort"
	"sync"
)

const (
	TYPE_CLOSE = 1
)

type Client struct {
	Id         string       // 客户端Id
	Conn       *net.TCPConn // 客户端连接

	InChan     chan (core.Frame)
	OutChan    chan (core.Frame)
	CloseChan  chan (int)

	channelIds []uint8      // 当前与客户端通信的通道Id列表

	IsClosed   bool
	IsAuthed   bool

	Lock       sync.RWMutex
}

func (this *Client) Close() {
	this.IsClosed = true
	this.Conn.Close()
	this.CloseChan <- TYPE_CLOSE
}

func NewClient(conn *net.TCPConn) *Client {
	c := &Client{
		Id:         conn.RemoteAddr().String(),
		Conn:conn,
		InChan:make(chan (core.Frame), 10000),
		OutChan:make(chan (core.Frame), 10000),
		CloseChan:make(chan (int), 100),
		channelIds:make([]uint8, 0),
		IsClosed:   false,
		IsAuthed:   false,
	}
	return c
}

// channel

func (this *Client) NewChannelId() uint8 {
	this.Lock.Lock()
	defer this.Lock.Unlock();
	sort.Slice(this.channelIds, func(i, j int) bool {
		return this.channelIds[i] - this.channelIds[j] < 0
	})
	for i := uint8(1); i < 255; i++ {
		used := false
		for _, id := range this.channelIds {
			if id == i {
				used = true
			}
		}
		if !used {
			this.channelIds = append(this.channelIds, i)
			return i
		}
	}
	this.channelIds = append(this.channelIds, 0)
	return 0
}

func (this *Client) RemoveChannelId(id uint8) {
	this.Lock.Lock()
	defer this.Lock.Unlock();
	for i, cId := range this.channelIds {
		if cId == id {
			this.channelIds = append(this.channelIds[:i], this.channelIds[i + 1:]...)
			break
		}
	}
}

func (this *Client) ConsistsChannelId(id uint8) bool {
	this.Lock.Lock()
	defer this.Lock.Unlock();
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
