package main

import (
	"os"

	"github.com/debdutdeb/novm/v3/commands"
	"github.com/debdutdeb/novm/v3/internal/log"
	"github.com/debdutdeb/novm/v3/utils"
)

func main() {
	if !utils.IsInteractive() {
		if err := commands.Run(); err != nil {
			log.Fatal(err)
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
