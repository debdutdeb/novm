package cmd

import (
	"fmt"

	"github.com/debdutdeb/node-proxy/versions"

	"github.com/spf13/cobra"
)

func versionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "version",
		Run: func(c *cobra.Command, args []string) {
			fmt.Printf("Version: %s\nGitCommit: %s\nBuildTime: %s\n", versions.Version, versions.GitCommit, versions.BuildTime)
		},
		Args: cobra.NoArgs,
	}

	return cmd
}
