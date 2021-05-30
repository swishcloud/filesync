package client_test

import (
	"os"
	"testing"
	"time"

	"github.com/swishcloud/filesync/internal"

	"github.com/swishcloud/filesync/client"
	"github.com/swishcloud/filesync/server"
)

func runSimpleServer() {
	server := server.NewFileSyncServer("server.yaml", true)
	go server.Serve()
	time.Sleep(time.Millisecond * 100)
}
func Test_Download_File(t *testing.T) {
	/*runSimpleServer()
	err := client.DownloadFile("fe59139b-9717-4795-94cf-65f0557ca177", true)
	if err != nil {
		t.Fatal(err)
	}*/
}

func Test_Send_File(t *testing.T) {
	runSimpleServer()
	err := client.SendFile("/root/Desktop/large file.zip", "", true)
	if err != nil {
		t.Fatal(err)
	}
}
func init() {
	internal.InitRAC(true)
	os.Chdir("../")
}
