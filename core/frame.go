package core

import (
	"encoding/binary"
	"io"
)

const (
	DATA  = 0
	BEGIN = 1
	DONE  = 2
	AUTH  = 3
)

type Frame []byte

func (f Frame) len() uint32 {
	return binary.BigEndian.Uint32(f)
}

func (f Frame) class() uint8 {
	return f[4]
}

func (f Frame) channel() uint8 {
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
func (fr *FrameReader) read() (f Frame, err error) {
	f = make(Frame, 6)
	_, err = io.ReadFull(fr.Reader, f)
	if err != nil {
		return
	}
	payloadSize := f.len() - 6
	if payloadSize > 0 {
		payload := make([]byte, payloadSize)
		_, err = io.ReadFull(fr.Reader, payload)
		if err != nil {
			return
		}
		xf := make(Frame, len(f))
		copy(xf, f)
		f = make(Frame, xf.len())
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

func (fw *FrameWriter) write(class, channel uint8, payload []byte) (n int, err error) {
	frame := NewFrame(class, channel, payload)
	return fw.Writer.Write(frame)
}

func (fw *FrameWriter) writeFrame(f Frame) (n int, err error) {
	return fw.Writer.Write(f)
}
