package main

import (
	"log"

	"github.com/debdutdeb/node-proxy/commands"
)

func main() {
	if err := handleNewInstall(); err != nil {
		log.Fatal("failed to run fresh install tasks: ", err)
	}

	cont, done := startCheckUpdate()

	if err := commands.Run(); err != nil {
		log.Fatal(err)
	}

	cont <- struct{}{}
	<-done
}
