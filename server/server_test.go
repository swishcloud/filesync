package server

import (
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/swishcloud/filesync/message"
)

func Test(t *testing.T) {
	b := []byte("p")

	fmt.Println(b)
}
func Test_message(t *testing.T) {
	msg := &message.Message{}
	msg.MsgType = 10
	msg.BodySize = 200
	r := msg.Reader()
	read, _ := message.ReadMessage(r)
	log.Println(read.MsgType)
}

func init() {
	os.Chdir("../")
}
