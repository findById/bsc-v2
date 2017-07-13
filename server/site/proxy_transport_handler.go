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
	for this.c != nil && !this.c.IsClosed && this.pc != nil && !this.pc.IsClosed {
		n, err := this.pc.Conn.Read(buf)
		if err != nil {
			log.Println("proxy read error", err)
			this.pc.Close()
			this.pcm.RemoveClient(this.pc.Id)

			// 通知客户端关闭数据通道
			data := core.NewFrame(core.CLOSE_CH, this.pc.ChannelId, core.NO_PAYLOAD)
			this.c.OutChan <- data
			this.c.RemoveChannelId(this.pc.ChannelId)
			// 通知客户端关闭连接
			if len(this.cm.CloneMap()) > 1 {
				if this.c.ChannelIdSize() <= 1 {
					data := core.NewFrame(core.CLOSE_CO, this.pc.ChannelId, core.NO_PAYLOAD)
					this.c.OutChan <- data
				}
			}
			return
		}
		log.Println("proxy read data")
		// 将数据处理权交给客户端连接处理
		data := core.NewFrame(core.DATA, this.pc.ChannelId, buf[:n])
		this.c.OutChan <- data
	}
}

func (this *SiteTransportHandler) WritePacket() {
	for this.c != nil && !this.c.IsClosed && this.pc != nil && !this.pc.IsClosed {
		select {
		case data := <-this.pc.OutChan:
			log.Println("proxy write data")
			this.pc.Conn.Write(data)
		}
	}
}
