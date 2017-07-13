package handler

import (
	"bsc-v2/server/client"
	"net"
	"bsc-v2/core"
	"bsc-v2/server/site"
	"log"
)

type TransportHandler struct {
	cm     *client.ClientManager
	client *client.Client
	pcm    *site.ProxyClientManager
}

func NewHandler(conn *net.TCPConn, cm *client.ClientManager, pcm *site.ProxyClientManager) *TransportHandler {
	client := client.NewClient(conn)
	return &TransportHandler{
		cm:      cm,
		client:  client,
		pcm:pcm,
	}
}

func (this *TransportHandler) Start() {
	go this.WritePacket()
	go this.ReadPacket()
}

func (this *TransportHandler) ReadPacket() {
	fr := core.NewFrameReader(this.client.Conn)
	for this.client != nil && !this.client.IsClosed {
		f, err := fr.Read()
		if err != nil {
			this.client.Close()
			return
		}
		log.Println("client read data", "channelId:", f.Channel(), "type:", core.RN[int(f.Class())])
		switch f.Class() {
		case core.AUTH:
			// 验证客户端后添加到连接库
			this.client.IsAuthed = true
			this.cm.AddClient(this.client)

			// 客户端认证回应
			log.Println("auth ack")
			this.client.OutChan <- core.NewFrame(core.AUTH_ACK, 0, []byte{0})
		case core.DATA:
			cId := f.Channel()
			// 查找是否存在当前channelId的连接，如该没有告诉客户端关闭数据通道
			c := this.pcm.GetClientByChannelId(cId, this.client.Id)
			if c == nil {
				// 通知客户端关闭当前数据通道
				log.Println("not found channel id", cId)
				this.client.OutChan <- core.NewFrame(core.CLOSE_CH, cId, core.NO_PAYLOAD)
				continue
			}
			// 把数据处理权交给对应channelId的用户连接
			c.OutChan <- f.Payload()
		}

	}
}

func (this *TransportHandler) WritePacket() {
	fw := core.NewFrameWriter(this.client.Conn)
	for this.client != nil && !this.client.IsClosed {
		select {
		case data := <-this.client.OutChan:
			log.Println("client write data", "channelId:", data.Channel(), "type:", core.RN[int(data.Class())])
			fw.WriteFrame(data)

		// 如果是关闭连接消息，发送后主动关掉连接
			if data.Class() == core.CLOSE_CO {
				this.cm.RemoveClient(this.client.Id)
			}
		}
	}
}
