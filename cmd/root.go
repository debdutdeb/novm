package cmd

import (
	"github.com/debdutdeb/novm/common"

	"github.com/spf13/cobra"
)

func Root(rootDir string) *cobra.Command {
	cmd := &cobra.Command{
		Use: common.BIN_NAME,
	}

	cmd.AddCommand(versionCommand())
	cmd.AddCommand(setupCommand(), whereCmd())

	return cmd
}
