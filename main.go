package main

import (
	"log"
	"os"

	"github.com/swishcloud/filesync/cmd"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.LUTC)
	cmd.Execute()
}
func init() {
	err := os.MkdirAll(".cache", os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}
}
