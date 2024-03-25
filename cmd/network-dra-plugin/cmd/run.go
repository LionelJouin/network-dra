package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/LionelJouin/network-dra/api/v1alpha1"
	"github.com/LionelJouin/network-dra/pkg/dra"
	"github.com/spf13/cobra"
)

type runOptions struct {
	driverPluginSocketPath string
	pluginRegistrationPath string
	cdiRoot                string
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
		&runOpts.nodeName,
		"node-name",
		"",
		"Node where the pod is running.",
	)

	return cmd
}

func (ro *runOptions) run(ctx context.Context) {
	draDriverName := v1alpha1.GroupName

	driver := dra.Driver{
		Name:                   v1alpha1.GroupName,
		DriverPluginPath:       filepath.Join(ro.driverPluginSocketPath, draDriverName),
		PluginRegistrationPath: filepath.Join(ro.pluginRegistrationPath, fmt.Sprintf("%s.sock", draDriverName)),
		CDIRoot:                ro.cdiRoot,
	}

	err := driver.Start(ctx)
	if err != nil {
		fmt.Printf("failed to start DRA driver: %v", err)
		os.Exit(1)
	}

}
