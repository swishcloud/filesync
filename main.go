package main

import (
	"log"

	"github.com/swishcloud/filesync/cmd"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.LUTC)
	cmd.Execute()
}
