package main

import (
	"log"

	"github.com/debdutdeb/node-proxy/commands"
)

func main() {
	cont, done := startCheckUpdate()

	if err := commands.Run(); err != nil {
		log.Fatal(err)
	}

	cont <- true
	<-done
}
