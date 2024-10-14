package cmd

import (
	"github.com/spf13/cobra"

	"github.com/debdutdeb/node-proxy/utils"
)

func setupCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "setup",
		RunE: func( c *cobra.Command, args []string) error {
			return utils.HandleNewInstall()
		},
	}

	return cmd
}
