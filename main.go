package main

import (
	"log"
	"os"

	"github.com/debdutdeb/node-proxy/commands"
	"github.com/debdutdeb/node-proxy/utils"
	"github.com/debdutdeb/node-proxy/versions"
)

func main() {
	if err := utils.HandleNewInstall(); err != nil {
		log.Fatal("failed to run fresh install tasks: ", err)
	}

	if versions.Version == "develop" {
		log.Printf("ignoring update check since is develop version\n")

		if err := commands.Run(); err != nil {
			log.Fatal(err)
			os.Exit(1)
		}

		return
	}

	if err := wrapInUpdateCheck(commands.Run); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}
