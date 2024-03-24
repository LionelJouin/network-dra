package cmd

import (
	"context"

	"github.com/spf13/cobra"
)

type runOptions struct{}

func newCmdRun() *cobra.Command {
	runOpts := &runOptions{}

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run the network-dra-controller",
		Long:  `Run the network-dra-controller`,
		Run: func(cmd *cobra.Command, args []string) {
			runOpts.run(cmd.Context())
		},
	}

	return cmd
}

func (ro *runOptions) run(ctx context.Context) {
}
