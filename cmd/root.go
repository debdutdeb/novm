package cmd

import (
	"github.com/debdutdeb/node-proxy/common"

	"github.com/spf13/cobra"
)

func Root(rootDir string) *cobra.Command {
	cmd := &cobra.Command{
		Use: common.BIN_NAME,
	}

	cmd.AddCommand(versionCommand())

	return cmd
}
