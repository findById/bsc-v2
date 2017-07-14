package handler

import (
	"bsc-v2/server/client"
	"net"
	"bsc-v2/core"
	"bsc-v2/server/site"
	"io"
	"log"
	"encoding/base64"
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
			// 读到结束符，不关闭客户端连接
			if err == io.EOF {
				continue
			}
			//log.Println("client read data error", err)
			if this.cm.Size() > 1 {
				this.cm.RemoveClient(this.client.Id)
			}
			return
		}
		//log.Printf("client read data >> cId:%s, chId:%d, t:%s \n", this.client.Id, int(f.Channel()), core.RN[int(f.Class())])
		switch f.Class() {
		case core.AUTH: // 客户端发起认证请求
			log.Printf("client read data >> cId:%s, chId:%d, t:%s \n", this.client.Id, int(f.Channel()), core.RN[int(f.Class())])
			// 验证客户端合法性
			b := base64.StdEncoding.EncodeToString(f.Payload())
			if b != this.cm.AuthToken {
				// 认证失败
				this.client.OutChan <- core.NewFrame(core.AUTH_ACK, 0, []byte{1})
				this.client.Close()
				return
			}
			// 认证成功, 添加到活动连接库
			this.client.IsAuthed = true
			this.cm.AddClient(this.client)

			this.client.OutChan <- core.NewFrame(core.AUTH_ACK, 0, []byte{0})
		case core.DATA: // 数据传输
			if !this.client.IsAuthed {
				if this.cm.Size() > 1 {
					this.cm.RemoveClient(this.client.Id)
				}
				return
			}
			cId := f.Channel()
			// 查找是否存在当前channelId的连接，如果没有告诉客户端关闭数据通道
			c := this.pcm.GetProxyClientByChannelId(cId, this.client.Id)
			if c == nil {
				// 通知客户端关闭当前数据通道
				log.Println("not found channel id", cId)
				this.client.OutChan <- core.NewFrame(core.CLOSE_CH, cId, core.NO_PAYLOAD)
				continue
			}
			// 把数据处理权交给对应channelId的用户连接
			log.Printf("write data %v, %v", c.Id, len(f.Payload()))
			c.OutChan <- f
		case core.CLOSE_CH: // 客户端发起的关闭通道请求
			log.Printf("client read data >> cId:%s, chId:%d, t:%s \n", this.client.Id, int(f.Channel()), core.RN[int(f.Class())])
			if !this.client.IsAuthed {
				if this.cm.Size() > 1 {
					this.cm.RemoveClient(this.client.Id)
				}
				return
			}
			this.client.RemoveChannelId(f.Channel())
			pc := this.pcm.GetProxyClientByChannelId(f.Channel(), this.client.Id)
			if pc != nil {
				pc.OutChan <- f
			}
		case core.CLOSE_CO: // 客户端发起的关闭连接请求
			log.Printf("client read data >> cId:%s, chId:%d, t:%s \n", this.client.Id, int(f.Channel()), core.RN[int(f.Class())])
			if !this.client.IsAuthed {
				if this.cm.Size() > 1 {
					this.cm.RemoveClient(this.client.Id)
				}
				return
			}
			if this.cm.Size() > 1 {
				this.cm.RemoveClient(this.client.Id)
			}
			pc := this.pcm.GetProxyClientByClientId(this.client.Id)
			if pc != nil {
				this.pcm.RemoveClient(pc.Id)
			}
		}
	}
}

func (this *TransportHandler) WritePacket() {
	fw := core.NewFrameWriter(this.client.Conn)
	for this.client != nil && !this.client.IsClosed {
		select {
		case data := <-this.client.OutChan:
			//log.Printf("client write data >> cId:%s, chId:%d, t:%s \n", this.client.Id, int(data.Channel()), core.RN[int(data.Class())])
			_, err := fw.WriteFrame(data)
			if err != nil {
				log.Println(err)
				this.cm.RemoveClient(this.client.Id)
				return
			}
		// 如果是关闭连接消息，发送后主动关掉连接
			if data.Class() == core.CLOSE_CO {
				this.cm.RemoveClient(this.client.Id)
			}
		case data := <-this.client.CloseChan:
			if data == client.TYPE_CLOSE {
				return
			}
		}
	}
}
