package message

import (
	"bytes"
	"encoding/gob"
	"io"
)

const (
	MT_PING           = 1
	MT_PANG           = 2
	MT_FILE           = 3
	MT_DISCONNECT     = 4
	MT_Request_Repeat = 5
	MT_Get_All_Files  = 6
	MT_Reply          = 7
	MT_Download_File  = 8
	MT_ACK            = 9
	MT_SYNC           = 10
)

type Message struct {
	Header   map[string]interface{}
	MsgType  int
	BodySize int64
}

func NewMessage(msgType int) *Message {
	msg := new(Message)
	msg.Header = make(map[string]interface{})
	msg.MsgType = msgType
	return msg
}
func (msg *Message) Reader() io.Reader {
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(msg)
	if err != nil {
		panic(err)
	}
	return buf
}
func ReadMessage(r io.Reader) (*Message, error) {
	dec := gob.NewDecoder(r)
	msg := new(Message)
	err := dec.Decode(msg)
	if err != nil {
		return nil, err
	}
	return msg, nil
}
