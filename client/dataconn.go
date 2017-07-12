package main

import (
	"io"
	"log"
	"net"
	"sync"
	//	"time"

	bsc "github.com/findById/bsc-v2/core"
)

type DataConn struct {
	targetAddr *net.TCPAddr
	serverAddr *net.TCPAddr
	conn       *net.TCPConn
	targets    map[uint8]*net.TCPConn // channel->target
	reader     *bsc.FrameReader
	exit       chan (int)
	lock       *sync.RWMutex
	//	rwLock     *sync.RWMutex
}

func NewDataConn(serverAddr, targetAddr *net.TCPAddr, exit chan (int)) *DataConn {
	return &DataConn{
		targetAddr: targetAddr,
		serverAddr: serverAddr,
		exit:       exit,
		targets:    make(map[uint8]*net.TCPConn),
		lock:       &sync.RWMutex{},
		//		rwLock:     &sync.RWMutex{},
	}
}

func (d *DataConn) closeChannel(ch uint8) {
	d.lock.Lock()
	defer d.lock.Unlock()
	log.Println("close channel", ch)
	if conn, ok := d.targets[ch]; ok {
		delete(d.targets, ch)
		log.Println("close target conn", conn.LocalAddr())
		conn.Close()
	}
}

func (d *DataConn) close() {
	d.lock.Lock()
	defer d.lock.Unlock()
	log.Println("close data conn", d.conn.LocalAddr())
	d.conn.Close()
	for ch, _ := range d.targets {
		d.closeChannel(ch)
	}
}

func (d *DataConn) read() (f bsc.Frame, err error) {
	//d.conn.SetReadDeadline(time.Now().Add(time.Second * 10))
	return d.reader.Read()
}

func (d *DataConn) do(ack bool) {
	defer func() {
		d.exit <- -1
	}()
	d.exit <- 1
	conn, err := net.DialTCP("tcp", nil, d.serverAddr)
	if err != nil {
		log.Println(err)
		return
	}
	conn.SetNoDelay(true)
	fw := bsc.NewFrameWriter(conn)
	fw.WriteUnPackFrame(bsc.AUTH, 0, []byte("hello bsc"))
	if ack {
		fw.WriteUnPackFrame(bsc.NEW_CO_ACK, 0, bsc.NO_PAYLOAD)
	}
	d.conn = conn
	d.reader = bsc.NewFrameReader(conn)
	for {
		frame, err := d.read()
		if err != nil {
			log.Println(err)
			break
		}
		log.Printf("new frame size: %d ,class: %s, channel:%d\n", frame.Size(), bsc.RN[int(frame.Class())], frame.Channel())
		if frame.Class() == bsc.DATA {
			if writer, ok := d.targets[frame.Channel()]; ok {
				_, err := writer.Write(frame.Payload())
				if err != nil {
					d.closeChannel(frame.Channel())
					log.Println(err)
					_, err := bsc.NewFrameWriter(d.conn).WriteUnPackFrame(bsc.CLOSE_CH, frame.Channel(), bsc.NO_PAYLOAD)
					if err != nil {
						d.close()
						log.Println(err)
						log.Println("close connection")
						break
					}
				}
			} else {
				go d.newChannel(frame.Channel(), frame.Payload())
			}
		} else if frame.Class() == bsc.AUTH_ACK {
			if frame.Payload()[0] != 0 {
				log.Println("auth failed")
				d.close()
				break
			}
		} else if frame.Class() == bsc.CLOSE_CH {
			log.Println("server request close channel", frame.Channel())
			d.closeChannel(frame.Channel())
		} else if frame.Class() == bsc.CLOSE_CO {
			log.Println("server request close connection")
			d.close()
			break
		} else if frame.Class() == bsc.NEW_CO {
			go NewDataConn(d.serverAddr, d.targetAddr, d.exit).do(true)
		} else {
			log.Println("not supported class", frame.Class())
		}
	}
}

func (d *DataConn) newChannel(ch uint8, payload []byte) {
	log.Println("new channel", ch)
	tConn, err := net.DialTCP("tcp", nil, d.targetAddr)
	if err != nil {
		log.Println(err)
		d.closeChannel(ch)
	}
	tConn.SetNoDelay(true)
	d.putTargets(ch, tConn)
	tConn.Write(payload)
	go func() {
		n, err := io.Copy(
			&bsc.FrameWriter{
				Channel: ch,
				Class:   bsc.DATA,
				Writer:  d.conn},
			tConn)
		log.Println("copy ", n, "bytes", err)
	}()
}

func (d *DataConn) putTargets(ch uint8, conn *net.TCPConn) {
	d.lock.Lock()
	defer d.lock.Unlock()
	d.targets[ch] = conn
}
