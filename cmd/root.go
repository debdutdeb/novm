package cmd

import "github.com/spf13/cobra"

func Root(rootDir string) *cobra.Command {
	cmd := &cobra.Command{
		Use: "node-proxy",
	}

	cmd.AddCommand(versionCommand())
	cmd.AddCommand(installCommand(rootDir))

	return cmd
}
