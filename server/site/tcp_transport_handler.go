package site

import (
	"bsc-v2/core"
	"bsc-v2/server/client"
	"log"
)

type TransportHandler struct {
	pc    *client.ProxyClient   // 代理客户端
	tc    *TcpClient            // 用户访问端
	cm    *client.ClientManager // 客户端连接管理
	tcm   *ClientManager        // 用户端连接管理
	debug bool
}

func NewSiteHandler(pc *client.ProxyClient, cm *client.ClientManager, tc *TcpClient, tcm *ClientManager, debug bool) *TransportHandler {
	return &TransportHandler{
		pc:    pc,
		cm:    cm,
		tc:    tc,
		tcm:   tcm,
		debug: debug,
	}
}

func (this *TransportHandler) Start() {
	go this.WritePacket()
	go this.ReadPacket()
}

func (this *TransportHandler) ReadPacket() {
	buf := make([]byte, 1024*8)
	for this.tc != nil && !this.tc.IsClosed {
		n, err := this.tc.Conn.Read(buf)
		//log.Printf("proxy read data >> %v", buf[:n])
		if this.debug {
			log.Printf("proxy read data >> %v", n)
		}
		// 如果有读到数据，将数据交给客户端连接处理
		if n > 0 {
			data := core.NewFrame(core.DATA, this.tc.ChannelId, buf[:n])
			this.pc.OutChan <- data
		}
		// 如果代理客户端已经关闭，就无法在提供服务
		if this.debug && (this.pc == nil || this.pc.IsClosed) {
			log.Println("client unavailable")
		}
		if err != nil || this.pc == nil || this.pc.IsClosed {
			//log.Println("proxy read data error", err)
			this.tcm.RemoveClient(this.tc.Id)

			// 通知客户端关闭数据通道
			if this.pc.ConsistsChannelId(this.tc.ChannelId) {
				data := core.NewFrame(core.CLOSE_CH, this.tc.ChannelId, core.NO_PAYLOAD)
				this.pc.OutChan <- data
			}

			// 如该当前与客户端的连接大于1个，寻找空闲的连接并通知客户端关闭 (设计重复,待优化 server.go)
			if len(this.cm.CloneMap()) > 1 {
				for _, v := range this.cm.CloneMap() {
					if v != this.pc && this.pc.ChannelIdSize() < 1 {
						data := core.NewFrame(core.CLOSE_CO, this.tc.ChannelId, core.NO_PAYLOAD)
						v.OutChan <- data
					}
				}
			}
			return
		}
	}
}

func (this *TransportHandler) WritePacket() {
	for this.tc != nil && !this.tc.IsClosed {
		select {
		case data := <-this.tc.OutChan:
			//log.Printf("proxy write data >> %v", data.Payload())
			if this.debug {
				log.Printf("proxy read data >> %v", data.Size()-core.FHL)
			}
			switch data.Class() {
			case core.DATA:
				_, err := this.tc.Conn.Write(data.Payload())
				if err != nil {
					this.tcm.RemoveClient(this.tc.Id)
					return
				}
			case core.CLOSE_CH:
				if data.Size() > core.FHL {
					this.tc.Conn.Write(data.Payload())
				}
				this.tcm.RemoveClient(this.tc.Id)
				// 响应释放通道成功
				this.pc.OutChan <- core.NewFrame(core.CLOSE_CH_ACK, this.tc.ChannelId, core.NO_PAYLOAD)
			}
		case data := <-this.tc.CloseChan:
			if data == TYPE_CLOSE {
				return
			}
		}
	}
}
