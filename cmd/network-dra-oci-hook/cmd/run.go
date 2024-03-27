package cmd

import (
	"context"
	"fmt"
	"io"
	"os"

	ociv1alpha1 "github.com/LionelJouin/network-dra/pkg/oci/api/v1alpha1"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type runOptions struct {
	claimUID          string
	OCIHookSocketPath string
}

func newCmdRun() *cobra.Command {
	runOpts := &runOptions{}

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run the network-dra-oci-hook",
		Long:  `Run the network-dra-oci-hook`,
		Run: func(cmd *cobra.Command, args []string) {
			runOpts.run(cmd.Context())
		},
	}

	cmd.Flags().StringVar(
		&runOpts.claimUID,
		"claim-uid",
		"",
		"Claim UID.",
	)

	cmd.Flags().StringVar(
		&runOpts.OCIHookSocketPath,
		"oci-hook-socket-path",
		"",
		"OCI hook socket path.",
	)

	return cmd
}

/*
#!/bin/bash
STD_IN=$(</dev/stdin)
echo "$STD_IN"
*/
func (ro *runOptions) run(ctx context.Context) {
	ociState, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading from stdin: %v", err)
		os.Exit(1)
	}

	conn, err := grpc.Dial(fmt.Sprintf("unix://%s", ro.OCIHookSocketPath),
		grpc.WithTransportCredentials(
			insecure.NewCredentials(),
		),
		grpc.WithBlock(),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error grpc.Dial: %v\n", err)
		os.Exit(1)
	}

	client := ociv1alpha1.NewOCIHookClient(conn)

	_, err = client.CreateContainer(ctx, &ociv1alpha1.CreateContainerRequest{
		Claim:    ro.claimUID,
		OciState: string(ociState),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error CreateContainer: %v\n", err)
		os.Exit(1)
	}
}
