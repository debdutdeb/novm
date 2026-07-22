package cmd

import (
	"fmt"
	"log"
	"runtime"

	"github.com/debdutdeb/novm/common"
	"github.com/debdutdeb/novm/pkg"
	"github.com/spf13/cobra"
)

func whereCmd() *cobra.Command {
	where := cobra.Command{
		Use:     "where",
		Aliases: []string{"which", "locate", "find"},
		Args:    cobra.ExactArgs(1),
		Short:   "where [version]",
		Long:    "get the location of installed version on disk",
		Run: func(cmd *cobra.Command, args []string) {
			version := args[0]
			n, err := pkg.NewNodeManager(false, version, common.RootDir)
			if err != nil {
				log.Fatal(err)
			}
			// FIXME: Version() code is weird, b ut i don't want to touch it right now. the len(xxx) == 0 { return string(xxx) } hain? anyway
			if n.Version() == "" {
				log.Fatalf("%s is not installed\n", version)
				return
			}

			fmt.Printf("%s/versions/%s/%s/%s\n", common.RootDir, n.Version(), runtime.GOOS, runtime.GOARCH)
		},
	}
	return &where
}
