package core

import (
	"encoding/binary"
	"io"
)

var RN = map[int]string{
	0: "数据",
	1: "打开新连接",
	2: "新链接ACK",
	3: "关闭通道",
	4: "关闭链接",
	5: "请求认证",
	6: "认证ACK",
	7: "PING",
	8: "PONG",
	9: "关闭通道ACK",
}

const (
	DATA         = 0 //数据 		  [L0,L1,L2,0,CHANNEL_ID,DATA....]
	NEW_CO       = 1 //打开新连接	  [L0,L1,L2,1,0]
	NEW_CO_ACK   = 2 //新链接ACK    [L0,L1,L2,2,0,REL] REL: 0 success,1 faild
	CLOSE_CH     = 3 //关闭通道     [L0,L1,L2,3,CHANNEL_ID]
	CLOSE_CO     = 4 //关闭链接     [L0,L1,L2,4,0]
	AUTH         = 5 //请求认证     [L0,L1,L2,5,0,DATA....] DATA:MD5(USER:PASSWD)
	AUTH_ACK     = 6 //认证ACK      [L0,L1,L2,6,0,REL] REL: 0 success,1 faild
	PING         = 7 //PING        [L0,L1,L2,7,0]
	PONG         = 8 //PONG        [L0,L1,L2,8,0]
	CLOSE_CH_ACK = 9 //关闭通道ACK	  [L0,L1,L2,9,0]
)

var NO_PAYLOAD = []byte{}
var AUTH_SUCCESS = []byte{0}
var AUTH_FAILED = []byte{1}

const (
	FHL = uint32(3 + 1 + 4)
	LI  = 0
	LL  = 3
	CI  = 3
	CHI = 4
	CHL = 4
)

type Frame []byte

func (f Frame) Size() uint32 {
	return binary.BigEndian.Uint32(f) >> 8
}

func (f Frame) Class() uint8 {
	return f[CI]
}

func (f Frame) Channel() uint32 {
	return binary.BigEndian.Uint32(f[CHI:])
}

func (f Frame) Payload() []byte {
	return f[FHL:]
}

func NewFrame(class uint8, channel uint32, payload []byte) (f Frame) {
	f = make(Frame, int(FHL)+len(payload))
	binary.BigEndian.PutUint32(f, uint32(len(f))<<8)
	f[CI] = class
	binary.BigEndian.PutUint32(f[CHI:], channel)
	copy(f[FHL:], payload)
	return
}

type FrameReader struct {
	Reader io.Reader
}

func NewFrameReader(r io.Reader) *FrameReader {
	return &FrameReader{Reader: r}
}
func (fr *FrameReader) Read() (f Frame, err error) {
	f = make(Frame, FHL)
	_, err = io.ReadFull(fr.Reader, f)
	if err != nil {
		return
	}
	payloadSize := f.Size() - FHL
	if payloadSize > 0 {
		xf := make(Frame, f.Size())
		copy(xf, f)
		_, err = io.ReadFull(fr.Reader, xf[FHL:])
		return xf, err
	}
	return
}

type FrameWriter struct {
	Writer  io.Writer
	Channel uint32
	Class   uint8
}

func NewFrameWriter(w io.Writer) *FrameWriter {
	return &FrameWriter{Writer: w}
}

func (fw *FrameWriter) WriteUnPackFrame(class uint8, channel uint32, payload []byte) (n int, err error) {
	frame := NewFrame(class, channel, payload)
	return fw.Writer.Write(frame)
}

func (fw *FrameWriter) Write(payload []byte) (n int, err error) {
	_, err = fw.WriteUnPackFrame(fw.Class, fw.Channel, payload)
	return len(payload), err
}

func (fw *FrameWriter) WriteFrame(f Frame) (n int, err error) {
	return fw.Writer.Write(f)
}
