package core

import (
	"encoding/binary"
	"io"
)

const (
	DATA       = 0 //数据 		 [L0,L1,L2,L3,0,CHANNEL_ID,DATA....]
	NEW_CO     = 1 //打开新连接	 [L0,L1,L2,L3,1,0]
	NEW_CO_ACK = 2 //新链接ACK     [L0,L1,L2,L3,2,0,REL] REL: 0 success,1 faild
	CLOSE_CH   = 3 //关闭通道		 [L0,L1,L2,L3,3,CHANNEL_ID]
	CLOSE_CO   = 4 //关闭链接		 [L0,L1,L2,L3,4,0]
	AUTH       = 5 //请求认证		 [L0,L1,L2,L3,5,0,DATA....] DATA:MD5(USER:PASSWD)
	AUTH_ACK   = 6 //认证ACK 		 [L0,L1,L2,L3,6,0,REL] REL: 0 success,1 faild
)

type Frame []byte

func (f Frame) Size() uint32 {
	return binary.BigEndian.Uint32(f)
}

func (f Frame) Class() uint8 {
	return f[4]
}

func (f Frame) Channel() uint8 {
	return f[5]
}

func (f Frame) Payload() []byte {
	return f[6:]
}

func NewFrame(class, channel uint8, payload []byte) (f Frame) {
	f = make(Frame, 6+len(payload))
	f[5] = channel
	f[4] = class
	binary.BigEndian.PutUint32(f, uint32(len(f)))
	copy(f[6:], payload)
	return
}

type FrameReader struct {
	Reader io.Reader
}

func NewFrameReader(r io.Reader) *FrameReader {
	return &FrameReader{Reader: r}
}
func (fr *FrameReader) Read() (f Frame, err error) {
	f = make(Frame, 6)
	_, err = io.ReadFull(fr.Reader, f)
	if err != nil {
		return
	}
	payloadSize := f.Size() - 6
	if payloadSize > 0 {
		payload := make([]byte, payloadSize)
		_, err = io.ReadFull(fr.Reader, payload)
		if err != nil {
			return
		}
		xf := make(Frame, len(f))
		copy(xf, f)
		f = make(Frame, xf.Size())
		copy(f, xf)
		copy(f[6:], payload)
	}
	return
}

type FrameWriter struct {
	Writer io.Writer
}

func NewFrameWriter(w io.Writer) *FrameWriter {
	return &FrameWriter{Writer: w}
}

func (fw *FrameWriter) Write(class, channel uint8, payload []byte) (n int, err error) {
	frame := NewFrame(class, channel, payload)
	return fw.Writer.Write(frame)
}

func (fw *FrameWriter) WriteFrame(f Frame) (n int, err error) {
	return fw.Writer.Write(f)
}
