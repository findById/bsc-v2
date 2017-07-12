package core

import (
	"bytes"
	"testing"
)

func TestFrame(t *testing.T) {
	x := []byte{0, 0, 0, 12, 1, 1, 'H', 'E', 'L', 'L', 'O', '!'}
	f := Frame(x)
	t.Logf("L:%d,C:%d,T:%d,P:%s", f.Size(), f.Channel(), f.Class(), string(f.Payload()))
	if f.Channel() != 1 || f.Class() != 1 || f.Size() != 12 || string(f.Payload()) != "HELLO!" {
		t.Fail()
	}
}

func TestWriter(t *testing.T) {
	var buf bytes.Buffer
	fw := NewFrameWriter(&buf)
	fw.write(1, 1, []byte("HELLO!"))
	f := Frame(buf.Bytes())
	t.Logf("L:%d,C:%d,T:%d,P:%s", f.Size(), f.Channel(), f.Class(), string(f.Payload()))
	if f.Channel() != 1 || f.Class() != 1 || f.Size() != 12 || string(f.Payload()) != "HELLO!" {
		t.Fail()
	}
}

func TestReader(t *testing.T) {
	buf := bytes.NewReader([]byte{0, 0, 0, 12, 1, 1, 'H', 'E', 'L', 'L', 'O', '!'})
	fr := NewFrameReader(buf)
	f, err := fr.read()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("L:%d,C:%d,T:%d,P:%s", f.Size(), f.Channel(), f.Class(), string(f.Payload()))
	if f.Channel() != 1 || f.Class() != 1 || f.Size() != 12 || string(f.Payload()) != "HELLO!" {
		t.Fail()
	}
}
