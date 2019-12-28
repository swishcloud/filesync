package client

import (
	"fmt"
	"log"
	"net"

	"github.com/swishcloud/filesync/message"

	"github.com/swishcloud/filesync/session"
)

func Ping(server string) {
	conn, err := net.Dial("tcp", server)
	if err != nil {
		log.Fatal(err)
	}
	s := session.NewSession(conn)
	msg := new(message.Message)
	msg.MsgType = message.MT_PING
	for i := 0; i < 5; i++ {
		err := s.Send(msg, nil)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(msg.MsgType)
		m, err := s.ReadMessage()
		if err != nil {
			log.Fatal(err)
		}
		if m.MsgType != message.MT_PANG {
			log.Fatal("unexpected response type:", m.MsgType)
		}
		fmt.Println(m.MsgType)
	}
	fmt.Println("server status is ok")
}
func SendFile(file_path string, server string) error {
	conn, err := net.Dial("tcp", server)
	if err != nil {
		log.Fatal(err)
	}
	s := session.NewSession(conn)
	err = s.SendFile(file_path)
	if err != nil {
		return err
	}
	return nil
}
