package site

import (
	"bsc-v2/server/client"
	"net"
	"bsc-v2/core"
	"time"
)

type SiteTransportHandler struct {
	cm      *client.ClientManager
	c       *client.Client

	pcm     *ProxyClientManager
	pc      *ProxyClient
	OutChan chan (core.Frame)
}

func NewSiteHandler(conn *net.TCPConn, pcm *ProxyClientManager, cm *client.ClientManager) *SiteTransportHandler {
	client := NewProxyClient(conn)
	return &SiteTransportHandler{
		pcm:  pcm,
		pc:  client,
		cm:      cm,
	}
}

func (this *SiteTransportHandler) Start() {
	beginTime := time.Now().Unix()
	a: // 未解决无客户端情况下接收到的连接
	for now := int64(0); (now - beginTime) < 10; now = time.Now().Unix() {
		// 查找可用连接通道 (临时解决方案)
		for _, conn := range this.cm.ConnMap {
			if conn.ChannelIdSize() < 200 {
				this.pc.ChannelId = conn.NewChannelId()
				this.c = conn // 复用当前可用数据通道
				break a
			} else {
				// 告诉客户端打开新的连接接收数据
				data := core.NewFrame(core.NEW_CO, this.pc.ChannelId, []byte(""))
				this.c.OutChan <- data

				// 等待客户端连接
				time.Sleep(100)
			}
		}
	}

	go this.WritePacket()
	go this.ReadPacket()
}

func (this *SiteTransportHandler) ReadPacket() {
	buf := make([]byte, 1024 * 8)
	for this.c != nil && !this.c.IsClosed && this.pc != nil && !this.pc.IsClosed {
		n, err := this.pc.Conn.Read(buf)
		if err != nil {
			data := core.NewFrame(core.CLOSE_CH, this.pc.ChannelId, core.NO_PAYLOAD)
			this.c.OutChan <- data
			this.pc.Close()
			return
		}

		// 将数据处理权交给客户端连接处理
		data := core.NewFrame(core.DATA, this.pc.ChannelId, buf[:n])
		this.c.OutChan <- data
	}
}

func (this *SiteTransportHandler) WritePacket() {
	for this.c != nil && !this.c.IsClosed && this.pc != nil && !this.pc.IsClosed {
		select {
		case data := <-this.pc.OutChan:
			this.pc.Conn.Write(data)
		}
	}
}
