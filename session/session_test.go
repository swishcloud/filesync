package session

import (
	"log"
	"net"
	"testing"

	"github.com/swishcloud/filesync/message"
)

func runServer(n int, c chan int64) {
	addr := ":3000"
	l, err := net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}
	log.Println("listening on", addr)
	c <- 1
	conn, err := l.Accept()
	if err != nil {
		panic(err)
	}
	s := NewSession(conn)
	for i := 0; i < n; i++ {
		msg, err := s.ReadMessage()
		if err != nil {
			panic(err)
		}
		var str = ""
		err = s.ReadJson(int(msg.BodySize), &str)
		if err != nil {
			panic(err)
		}
		log.Println(i, str)
	}
	msg, err := s.ReadMessage()
	if err != nil {
		panic(err)
	}
	filepath := "/root/Desktop/root1/test function generated"
	written, err := s.ReadFile(filepath, msg.Header["md5"].(string), msg.BodySize)
	if err != nil {
		panic(err)
	}
	log.Println("received file:", msg.Header["file_name"])
	if err != nil {
		panic(err)
	}
	if written != msg.BodySize {
		panic("size error")
	}
	msg, err = s.ReadMessage()
	if err != nil {
		panic(err)
	}
	log.Println(msg.Header["talk"])
	reply := message.NewMessage(2)
	reply.Header["talk"] = "I'm fine!fk u"
	err = s.SendMessage(reply, nil, 0)
	if err != nil {
		panic(err)
	}
	msg, err = s.ReadMessage()
	if err != nil {
		panic(err)
	}
	log.Println(msg.Header["talk"])
	reply.Header["talk"] = "Goodbye"
	err = s.SendMessage(reply, nil, 0)
	if err != nil {
		panic(err)
	}
	_, err = s.ReadMessage()
	if err != nil {
		log.Println("client connection disconnected")
		s.Close()
	}
	c <- s.read
}
func Test_Session(t *testing.T) {
	addr := ":3000"
	c := make(chan int64)
	n := 3000
	go runServer(n, c)
	<-c
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	s := NewSession(conn)
	for i := 0; i < n; i++ {
		msg := message.NewMessage(message.MT_PING)
		err = s.Send(msg, "hello world")
		if err != nil {
			log.Fatal(err)
		}
	}
	flepath := "/root/Desktop/root1/F/large file.zip"
	err = s.SendFile(flepath, nil)
	if err != nil {
		log.Fatal(err)
	}
	msg := message.NewMessage(message.MT_PING)
	msg.Header["talk"] = "how are you"
	reply, err := s.Fetch(msg, nil)
	if err != nil {
		panic(err)
	}
	log.Println(reply.Header["talk"])
	msg.Header["talk"] = "Goodbye"
	reply, err = s.Fetch(msg, nil)
	if err != nil {
		panic(err)
	}
	log.Println(reply.Header["talk"])
	s.Close()
	s_r := <-c
	log.Printf("client sent %d bytes to server,server received %d bytes", s.written, s_r)
	if s_r != s.written {
		t.FailNow()
	}
}
