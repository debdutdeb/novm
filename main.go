package main

import (
	"log"

	"github.com/debdutdeb/node-proxy/commands"
)

var NodeJsVersion string = ""

func main() {
	cont, done := startCheckUpdate()

	if err := commands.Run(); err != nil {
		log.Fatal(err)
	}

	cont <- true
	<-done
}
