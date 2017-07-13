package main

import (
	"io"
	"log"
	"net"
	"sync"

	bsc "github.com/findById/bsc-v2/core"
)

var (
	atlk  = sync.RWMutex{}
	idGen = int64(0)
)

func nextId() int64 {
	atlk.Lock()
	defer atlk.Unlock()
	idGen++
	return idGen
}

func getId() int64 {
	atlk.RLock()
	defer atlk.RUnlock()
	return idGen
}

type DataConn struct {
	id         int64
	token      []byte
	targetAddr *net.TCPAddr
	serverAddr *net.TCPAddr
	conn       *net.TCPConn
	targets    map[uint8]*net.TCPConn
	reader     *bsc.FrameReader
	exit       chan (int)
	lock       *sync.Mutex
}

func NewDataConn(serverAddr, targetAddr *net.TCPAddr, token []byte, exit chan (int)) *DataConn {
	return &DataConn{
		id:         nextId(),
		token:      token,
		targetAddr: targetAddr,
		serverAddr: serverAddr,
		exit:       exit,
		targets:    make(map[uint8]*net.TCPConn),
		lock:       &sync.Mutex{},
	}
}

func (d *DataConn) closeChannel(ch uint8, notify bool) {
	d.lock.Lock()
	defer d.lock.Unlock()
	if notify {
		bsc.NewFrameWriter(d.conn).WriteUnPackFrame(bsc.CLOSE_CH, ch, bsc.NO_PAYLOAD)
	}
	//d.logf("close channel %d", ch)
	if conn, ok := d.targets[ch]; ok {
		delete(d.targets, ch)
		//		d.logf("close target conn %s", conn.LocalAddr().String())
		conn.Close()
	}
}

func (d *DataConn) close() {
	d.lock.Lock()
	defer d.lock.Unlock()
	if d.conn != nil {
		//d.logf("close data conn %s", d.conn.LocalAddr().String())
		d.conn.Close()
	}
	for ch, _ := range d.targets {
		//d.logf("close channel %d", ch)
		if conn, ok := d.targets[ch]; ok {
			delete(d.targets, ch)
			//d.logf("close target conn %s", conn.LocalAddr().String())
			conn.Close()
		}
	}
}

func (d *DataConn) read() (f bsc.Frame, err error) {
	return d.reader.Read()
}
func (d *DataConn) findTarget(ch uint8) (conn *net.TCPConn, ok bool) {
	d.lock.Lock()
	defer d.lock.Unlock()
	if conn, ok = d.targets[ch]; ok {
		return conn, ok
	}
	return nil, false
}
func (d *DataConn) do(ack bool) {
	defer func(exit chan (int), id int64) {
		exit <- -1
		log.Printf("[%d] JOB DONE.", id)
	}(d.exit, d.id)
	defer d.close()

	d.exit <- 1
	conn, err := net.DialTCP("tcp", nil, d.serverAddr)
	if err != nil {
		d.logf("dial server err:%v", err)
		return
	}
	conn.SetNoDelay(true)
	fw := bsc.NewFrameWriter(conn)
	fw.WriteUnPackFrame(bsc.AUTH, 0, d.token)
	if ack {
		fw.WriteUnPackFrame(bsc.NEW_CO_ACK, 0, bsc.NO_PAYLOAD)
	}
	d.conn = conn
	d.reader = bsc.NewFrameReader(conn)
	for {
		frame, err := d.read()
		if err != nil {
			d.logf("read server err:%v", err)
			break
		}
		if frame.Class() != bsc.DATA && frame.Class() != bsc.PING {
			d.logf("new frame size: %d ,class: %s, channel:%d", frame.Size(), bsc.RN[int(frame.Class())], frame.Channel())
		}
		if frame.Class() == bsc.DATA {
			if writer, ok := d.findTarget(frame.Channel()); ok {
				_, err := writer.Write(frame.Payload())
				if err != nil {
					d.closeChannel(frame.Channel(), false)
					d.logf("write target with err: %v", err)
					_, err := bsc.NewFrameWriter(d.conn).WriteUnPackFrame(bsc.CLOSE_CH, frame.Channel(), bsc.NO_PAYLOAD)
					if err != nil {
						d.logf("close connection with err:%v", err)
						break
					}
				}
			} else {
				go d.newChannel(frame.Channel(), frame.Payload())
			}
		} else if frame.Class() == bsc.AUTH_ACK {
			if frame.Payload()[0] != 0 {
				d.logf("auth failed")
				break
			}
		} else if frame.Class() == bsc.CLOSE_CH {
			d.logf("server request close channel %d", frame.Channel())
			d.closeChannel(frame.Channel(), false)
		} else if frame.Class() == bsc.CLOSE_CO {
			d.logf("server request close connection")
			break
		} else if frame.Class() == bsc.NEW_CO {
			go NewDataConn(d.serverAddr, d.targetAddr, d.token, d.exit).do(true)
		} else if frame.Class() == bsc.PING {
			_, err := bsc.NewFrameWriter(d.conn).WriteUnPackFrame(bsc.PONG, 0, bsc.NO_PAYLOAD)
			if err != nil {
				d.logf("close connection with err:%v", err)
				break
			}
		}
	}
}

func (d *DataConn) newChannel(ch uint8, payload []byte) {
	//d.logf("new channel %d", ch)
	tConn, err := net.DialTCP("tcp", nil, d.targetAddr)
	if err != nil {
		d.logf("dial target err:%v", err)
		d.closeChannel(ch, true)
		return
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
		if err != nil {
			d.logf("copy %d bytes, with err:%v", n, err)
		}
		d.closeChannel(ch, true)
	}()
}

func (d *DataConn) putTargets(ch uint8, conn *net.TCPConn) {
	d.lock.Lock()
	defer d.lock.Unlock()
	d.targets[ch] = conn
}

func (d *DataConn) logf(format string, v ...interface{}) {
	vars := make([]interface{}, 1+len(v))
	vars[0] = d.id
	copy(vars[1:], v)
	log.Printf("[%d] "+format+"\n", vars...)
}
