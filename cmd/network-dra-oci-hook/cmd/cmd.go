package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by the main function.
func Execute() {
	if err := getRootCmd().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func getRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "network-dra-oci-hook",
		Short: "CLI",
		Long:  `CLI for interacting with the network-dra-oci-hook`,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
	}

	rootCmd.AddCommand(newCmdRun())

	return rootCmd
}
