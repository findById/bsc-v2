package main

import (
	"errors"
	"io"
	"log"
	"net"
	"sync"

	bsc "github.com/findById/bsc-v2/core"
)

var (
	atlk       = sync.RWMutex{}
	idGen      = int64(0)
	AUTH_FAILD = errors.New("auth failed")
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
	id             int64
	token          []byte
	targetAddr     *net.TCPAddr
	serverAddr     *net.TCPAddr
	conn           *net.TCPConn
	targets        map[uint8]*net.TCPConn
	connMonitor    *chan (int)
	channelMonitor *chan (int)
	lock           *sync.Mutex
	debug          bool
	nodelay        bool
}

func NewDataConn(serverAddr, targetAddr *net.TCPAddr, token []byte, nodelay, debug bool, cm, chm *chan (int)) *DataConn {
	return &DataConn{
		id:             nextId(),
		token:          token,
		debug:          debug,
		nodelay:        nodelay,
		targetAddr:     targetAddr,
		serverAddr:     serverAddr,
		connMonitor:    cm,
		channelMonitor: chm,
		targets:        make(map[uint8]*net.TCPConn),
		lock:           &sync.Mutex{},
	}
}

func (d *DataConn) closeChannel(ch uint8, notify bool) {
	d.lock.Lock()
	defer d.lock.Unlock()
	if notify {
		bsc.NewFrameWriter(d.conn).WriteUnPackFrame(bsc.CLOSE_CH, ch, bsc.NO_PAYLOAD)
	}
	d.logf("close channel %d", ch)
	if conn, ok := d.targets[ch]; ok {
		if d.channelMonitor != nil {
			*d.channelMonitor <- -1
		}
		delete(d.targets, ch)
		d.logf("close target conn %s", conn.LocalAddr().String())
		conn.Close()
	}
}

func (d *DataConn) findTarget(ch uint8) (conn *net.TCPConn, ok bool) {
	d.lock.Lock()
	defer d.lock.Unlock()
	if conn, ok = d.targets[ch]; ok {
		return conn, ok
	}
	return nil, false
}

func (d *DataConn) putTargets(ch uint8, conn *net.TCPConn) {
	d.lock.Lock()
	defer d.lock.Unlock()
	d.targets[ch] = conn
}

func (d *DataConn) close() {
	d.lock.Lock()
	defer d.lock.Unlock()
	if d.conn != nil {
		d.logf("close data conn %s", d.conn.LocalAddr().String())
		d.conn.Close()
	}
	for ch, _ := range d.targets {
		d.logf("close channel %d", ch)
		if conn, ok := d.targets[ch]; ok {
			delete(d.targets, ch)
			d.logf("close target conn %s", conn.LocalAddr().String())
			conn.Close()
		}
	}
}

func (d *DataConn) do(ack bool) (xerr error) {
	defer func(cm *chan (int), id int64) {
		if cm != nil {
			*cm <- -1
		}
		log.Printf("[%d] JOB DONE.", id)
	}(d.connMonitor, d.id)
	defer d.close()
	if d.connMonitor != nil {
		*d.connMonitor <- 1
	}
	conn, err := net.DialTCP("tcp", nil, d.serverAddr)
	if err != nil {
		d.logf("dial server err:%v", err)
		return
	}
	conn.SetNoDelay(d.nodelay)
	fw := bsc.NewFrameWriter(conn)
	_, err = fw.WriteUnPackFrame(bsc.AUTH, 0, d.token)
	if err != nil {
		d.logf("auth failed: %v", err)
		return
	}
	if ack {
		_, err = fw.WriteUnPackFrame(bsc.NEW_CO_ACK, 0, bsc.NO_PAYLOAD)
		if err != nil {
			d.logf("new connection ack failed: %v", err)
			return
		}
	}
	d.conn = conn
	reader := bsc.NewFrameReader(conn)
	for {
		frame, err := reader.Read()
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
						d.logf("close connection with err: %v", err)
						break
					}
				}
			} else {
				go d.newChannel(frame.Channel(), frame.Payload())
			}
		} else if frame.Class() == bsc.AUTH_ACK {
			if frame.Payload()[0] != 0 {
				d.logf("auth failed")
				xerr = AUTH_FAILD
				break
			}
		} else if frame.Class() == bsc.CLOSE_CH {
			d.closeChannel(frame.Channel(), false)
			_, err := bsc.NewFrameWriter(d.conn).WriteUnPackFrame(bsc.CLOSE_CH_ACK, frame.Channel(), bsc.NO_PAYLOAD)
			if err != nil {
				d.logf("close connection with err: %v", err)
				break
			}
		} else if frame.Class() == bsc.CLOSE_CO {
			d.logf("server request close connection")
			break
		} else if frame.Class() == bsc.NEW_CO {
			go NewDataConn(d.serverAddr, d.targetAddr, d.token, d.nodelay, d.debug, d.connMonitor, d.channelMonitor).do(true)
		} else if frame.Class() == bsc.PING {
			_, err := bsc.NewFrameWriter(d.conn).WriteUnPackFrame(bsc.PONG, 0, bsc.NO_PAYLOAD)
			if err != nil {
				d.logf("close connection with err:%v", err)
				break
			}
		}
	}
	return
}

func (d *DataConn) newChannel(ch uint8, payload []byte) {
	if d.channelMonitor != nil {
		*d.channelMonitor <- 1
	}
	d.logf("new channel %d ,with %d byte payload", ch, len(payload))
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
		d.logf("copy %d bytes, with err:%v", n, err)
		d.closeChannel(ch, true)
	}()
}

func (d *DataConn) logf(format string, v ...interface{}) {
	if d.debug {
		vars := make([]interface{}, 1+len(v))
		vars[0] = d.id
		copy(vars[1:], v)
		log.Printf("[%d] "+format+"\n", vars...)
	}
}
