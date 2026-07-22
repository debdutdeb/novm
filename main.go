package main

import (
	"log"
	"os"

	"github.com/debdutdeb/novm/commands"
	"github.com/debdutdeb/novm/utils"
)

func main() {
	if !utils.IsInteractive() {
		if err := commands.Run(); err != nil {
			log.Fatal(err)
			os.Exit(1)
		}
		return
	}

	if err := utils.HandleNewInstall(); err != nil {
		log.Fatal("failed to run fresh install tasks: ", err)
	}

	if err := wrapInUpdateCheck(commands.Run); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}
