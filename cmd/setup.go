package cmd

import (
	"github.com/spf13/cobra"

	"github.com/debdutdeb/novm/v3/utils"
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
