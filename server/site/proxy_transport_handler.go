package site

import (
	"bsc-v2/server/client"
	"bsc-v2/core"
	"log"
)

type SiteTransportHandler struct {
	c   *client.Client        // 代理客户端
	pc  *ProxyClient          // 用户访问端
	cm  *client.ClientManager // 客户端连接管理
	pcm *ProxyClientManager   // 用户端连接管理
}

func NewSiteHandler(c *client.Client, cm *client.ClientManager, pc *ProxyClient, pcm *ProxyClientManager) *SiteTransportHandler {
	return &SiteTransportHandler{
		c:c,
		cm:cm,
		pc:  pc,
		pcm:pcm,
	}
}

func (this *SiteTransportHandler) Start() {
	go this.WritePacket()
	go this.ReadPacket()
}

func (this *SiteTransportHandler) ReadPacket() {
	buf := make([]byte, 1024 * 8)
	for this.pc != nil && !this.pc.IsClosed {
		n, err := this.pc.Conn.Read(buf)
		//log.Printf("proxy read data >> %v", buf[:n])
		// 如果有读到数据，将数据交给客户端连接处理
		if n > 0 {
			data := core.NewFrame(core.DATA, this.pc.ChannelId, buf[:n])
			this.c.OutChan <- data
		}
		// 如果代理客户端已经关闭，就无法在提供服务
		if err != nil || this.c == nil || this.c.IsClosed {
			log.Println("proxy read data error", err)
			this.pcm.RemoveClient(this.pc.Id)

			// 通知客户端关闭数据通道
			if this.c.ConsistsChannelId(this.pc.ChannelId) {
				this.c.RemoveChannelId(this.pc.ChannelId)
				data := core.NewFrame(core.CLOSE_CH, this.pc.ChannelId, core.NO_PAYLOAD)
				this.c.OutChan <- data
			}

			// 如该当前与客户端的连接大于1个，寻找空闲的连接并通知客户端关闭 (设计重复,待优化 server.go)
			if len(this.cm.CloneMap()) > 1 {
				for _, v := range this.cm.CloneMap() {
					if v != this.c && this.c.ChannelIdSize() < 1 {
						data := core.NewFrame(core.CLOSE_CO, this.pc.ChannelId, core.NO_PAYLOAD)
						v.OutChan <- data
					}
				}
			}
			return
		}
	}
}

func (this *SiteTransportHandler) WritePacket() {
	for this.pc != nil && !this.pc.IsClosed {
		select {
		case data := <-this.pc.OutChan:
		//log.Printf("proxy write data >> %v", data.Payload())
			switch data.Class() {
			case core.DATA:
				_, err := this.pc.Conn.Write(data.Payload())
				if err != nil {
					this.pcm.RemoveClient(this.pc.Id)
					return
				}
			case core.CLOSE_CH:
				this.pcm.RemoveClient(this.pc.Id)
			}
		case data := <-this.pc.CloseChan:
			if data == TYPE_CLOSE {
				return
			}
		}
	}
}
