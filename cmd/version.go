package cmd

import (
	"fmt"
	"time"

	"github.com/debdutdeb/node-proxy/versions"

	"github.com/spf13/cobra"
)

func versionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "version",
		Run: func(c *cobra.Command, args []string) {
			buildTime, _ := time.Parse(time.UnixDate, versions.BuildTime)

			fmt.Printf("Version: %s\nGitCommit: %s\nBuildTime: %s\n", versions.Version, versions.GitCommit, buildTime.In(time.FixedZone(time.Now().Zone())).Format(time.UnixDate))
		},
		Args: cobra.NoArgs,
	}

	return cmd
}
