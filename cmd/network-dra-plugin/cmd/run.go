package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/LionelJouin/network-dra/api/dra.networking/v1alpha1"
	"github.com/LionelJouin/network-dra/pkg/dra"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type runOptions struct {
	driverPluginSocketPath string
	pluginRegistrationPath string
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
		&runOpts.nodeName,
		"node-name",
		"",
		"Node where the pod is running.",
	)

	return cmd
}

func (ro *runOptions) run(ctx context.Context) {
	draDriverName := v1alpha1.GroupName

	clientCfg, err := rest.InClusterConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to InClusterConfig: %v\n", err)
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(clientCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to NewForConfig: %v\n", err)
		os.Exit(1)
	}

	driver := dra.Driver{
		Name:                   v1alpha1.GroupName,
		DriverPluginPath:       filepath.Join(ro.driverPluginSocketPath, draDriverName),
		PluginRegistrationPath: filepath.Join(ro.pluginRegistrationPath, fmt.Sprintf("%s.sock", draDriverName)),
		NodeName:               ro.nodeName,
		Clientset:              clientset,
	}

	err = driver.Start(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start DRA plugin: %v\n", err)
		os.Exit(1)
	}
}
