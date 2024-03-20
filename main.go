package main

import (
	"log"

	"github.com/debdutdeb/node-proxy/commands"
)

var NodeJsVersion string = ""

func main() {
	if err := commands.Run(); err != nil {
		log.Fatal(err)
	}
}

