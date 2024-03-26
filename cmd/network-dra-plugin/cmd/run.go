package cmd

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/LionelJouin/network-dra/api/v1alpha1"
	"github.com/LionelJouin/network-dra/pkg/dra"
	ociv1alpha1 "github.com/LionelJouin/network-dra/pkg/oci/api/v1alpha1"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

type runOptions struct {
	driverPluginSocketPath string
	pluginRegistrationPath string
	cdiRoot                string
	OCIHookPath            string
	nodeName               string
}

func newCmdRun() *cobra.Command {
	runOpts := &runOptions{}

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run the network-dra-plugin",
		Long:  `Run the network-dra-plugin`,
		Run: func(cmd *cobra.Command, args []string) {
			runOpts.run(cmd.Context())
		},
	}

	cmd.Flags().StringVar(
		&runOpts.driverPluginSocketPath,
		"driver-plugin-path",
		"/var/lib/kubelet/plugins/",
		"Path to the driver plugin directory.",
	)

	cmd.Flags().StringVar(
		&runOpts.pluginRegistrationPath,
		"plugin-registration-path",
		"/var/lib/kubelet/plugins_registry/",
		"Path to the registration plugin directory.",
	)

	cmd.Flags().StringVar(
		&runOpts.cdiRoot,
		"cdi-root",
		"/var/run/cdi",
		"Path to the cdi files directory.",
	)

	cmd.Flags().StringVar(
		&runOpts.OCIHookPath,
		"oci-hook-path",
		"/network-dra-plugin-oci-hook/",
		"oci hook path.",
	)

	cmd.Flags().StringVar(
		&runOpts.nodeName,
		"node-name",
		"",
		"Node where the pod is running.",
	)

	return cmd
}

func (ro *runOptions) run(ctx context.Context) {
	draDriverName := v1alpha1.GroupName
	ociHookSocketPath := filepath.Join(ro.OCIHookPath, "oci-hook-callback.sock")

	driver := dra.Driver{
		Name:                   v1alpha1.GroupName,
		DriverPluginPath:       filepath.Join(ro.driverPluginSocketPath, draDriverName),
		PluginRegistrationPath: filepath.Join(ro.pluginRegistrationPath, fmt.Sprintf("%s.sock", draDriverName)),
		CDIRoot:                ro.cdiRoot,
		OCIHookPath:            filepath.Join(ro.OCIHookPath, "network-dra-oci-hook"),
		OCIHookSocketPath:      ociHookSocketPath,
	}

	if err := os.RemoveAll(ociHookSocketPath); err != nil {
		fmt.Fprintf(os.Stderr, "failed to remove socket: %v\n", err)
		os.Exit(1)
	}

	lis, err := net.Listen("unix", ociHookSocketPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to listen: %v\n", err)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer()

	hookCallbackServer := &dra.OCIHookCallbackServer{}

	go func() {
		ociv1alpha1.RegisterOCIHookServer(grpcServer, hookCallbackServer)

		err = grpcServer.Serve(lis)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to serve: %v\n", err)
			os.Exit(1)
		}
	}()

	err = driver.Start(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start DRA driver: %v\n", err)
		os.Exit(1)
	}

	grpcServer.Stop()
}
