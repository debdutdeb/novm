package main

import (
	"log"

	"github.com/debdutdeb/node-proxy/commands"
	"github.com/debdutdeb/node-proxy/utils"
)

func main() {
	if err := utils.HandleNewInstall(); err != nil {
		log.Fatal("failed to run fresh install tasks: ", err)
	}

	cont, done := startCheckUpdate()

	if err := commands.Run(); err != nil {
		log.Fatal(err)
	}

	cont <- struct{}{}
	<-done
}
