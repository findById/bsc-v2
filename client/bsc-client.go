package main

import (
	"flag"
	"log"
	"net"
	"os"

	bsc "github.com/findById/bsc-v2/core"
)

var (
	server      = flag.String("s", "", "server address")
	target      = flag.String("t", "", "target service address")
	targets     = make(map[uint8]net.Conn)   // channel->target
	dataConns   = make(map[uint8]net.Conn)   // channel->server
	connChannel = make(map[net.Conn][]uint8) // server -> channel

)

func closeChannel(ch uint8) {
	if conn, ok := targets[ch]; ok {
		delete(targets, ch)
		conn.Close()
	}
	delete(dataConns, ch)
}

func closeConn(conn net.Conn) {
	conn.Close()
	if chs, ok := connChannel[conn]; ok {
		for _, ch := range chs {
			closeChannel(ch)
		}
		delete(connChannel, conn)
	}
}

func NewChannel(channel uint8, frameWriter *bsc.FrameWriter) {

}

func main() {
	flag.Parse()
	if *server == "" || *target == "" {
		flag.PrintDefaults()
		return
	}
	serverAddr, err := net.ResolveTCPAddr("tcp", *server)
	if err != nil {
		log.Println(err)
		return
	}
	conn, err := net.DialTCP("tcp", nil, serverAddr)
	if err != nil {
		log.Println(err)
		return
	}
	exit := make(chan (int))
	go func() {
		reader := bsc.NewFrameReader(conn)
		for {
			frame, err := reader.Read()
			if err != nil {
				log.Println(err)
				break
			}
			if frame.Class() == bsc.DATA {
				if writer, ok := targets[frame.Channel()]; ok {
					_, err := writer.Write(frame.Payload())
					if err != nil {
						closeChannel(frame.Channel())
						log.Println(err)
						log.Println("close channel", frame.Channel())
						if sWriter, ok := dataConns[frame.Channel()]; ok {
							_, err := bsc.NewFrameWriter(sWriter).Write(bsc.CLOSE_CH, frame.Channel(), bsc.NO_PAYLOAD)
							if err != nil {
								closeConn(sWriter)
								log.Println(err)
								log.Println("close connection")
							}
						}
					}
				} else {
					go NewChannel(frame.Channel(), bsc.NewFrameWriter(conn))
				}
			} else if frame.Class() == bsc.AUTH_ACK {
				if frame.Payload()[0] != 0 {
					log.Println("auth failed")
					os.Exit(0)
				}
			} else if frame.Class() == bsc.CLOSE_CH {

			} else if frame.Class() == bsc.CLOSE_CO {

			} else if frame.Class() == bsc.NEW_CO {

			} else {
				log.Println("not supported class", frame.Class())
			}

		}
		exit <- 1
	}()
	<-exit
}
