package protocol

type DataFrame struct {
	Length    int32  // BE
	Type      int8   // 1 byte (0 data, 1 connect, 2 close)
	ChannelId int8   // 0 ~ 255
	Data      []byte // ...
}