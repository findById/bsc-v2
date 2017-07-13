package site

import (
	"bsc-v2/server/client"
	"bsc-v2/core"
	"log"
)

type SiteTransportHandler struct {
	c       *client.Client // 代理客户端
	pc      *ProxyClient   // 用户访问端
	OutChan chan (core.Frame)
}

func NewSiteHandler(c *client.Client, pc *ProxyClient) *SiteTransportHandler {
	return &SiteTransportHandler{
		c:c,
		pc:  pc,
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
			data := core.NewFrame(core.CLOSE_CH, this.pc.ChannelId, core.NO_PAYLOAD)
			this.c.OutChan <- data
			this.pc.Close()
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
