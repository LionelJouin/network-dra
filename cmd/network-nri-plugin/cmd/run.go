package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/LionelJouin/network-dra/pkg/cni"
	"github.com/LionelJouin/network-dra/pkg/nri"
	"github.com/containerd/nri/pkg/api"
	"github.com/containerd/nri/pkg/stub"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type runOptions struct {
	pluginName  string
	pluginIndex string
	CNIPath     string
	CNICacheDir string
	ChrootDir   string
}

func newCmdRun() *cobra.Command {
	runOpts := &runOptions{}

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run the network-nri-plugin",
		Long:  `Run the network-nri-plugin`,
		Run: func(cmd *cobra.Command, args []string) {
			runOpts.run(cmd.Context())
		},
	}

	cmd.Flags().StringVar(
		&runOpts.pluginName,
		"plugin-name",
		"network-nri-plugin",
		"Plugin name to register to NRI.",
	)

	cmd.Flags().StringVar(
		&runOpts.pluginIndex,
		"plugin-index",
		"",
		"plugin index to register to NRI.",
	)

	cmd.Flags().StringVar(
		&runOpts.CNIPath,
		"cni-path",
		"/opt/cni/bin",
		"CNI Path.",
	)

	cmd.Flags().StringVar(
		&runOpts.CNICacheDir,
		"cni-cache-dir",
		"/var/lib/cni/nri-network",
		"CNI Cache dir.",
	)
	cmd.Flags().StringVar(
		&runOpts.ChrootDir,
		"chroot-dir",
		"/hostroot",
		"ChrootDir.",
	)

	return cmd
}

func (ro *runOptions) run(ctx context.Context) {
	opts := []stub.Option{
		stub.WithPluginName(ro.pluginName),
		stub.WithPluginIdx(ro.pluginIndex),
	}

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

	p := &nri.Plugin{
		ClientSet: clientset,
		Exec: &cni.ChrootExec{
			Stderr:    os.Stderr,
			ChrootDir: ro.ChrootDir,
		},
		CNIPath:     []string{ro.CNIPath},
		CNICacheDir: ro.CNICacheDir,
	}

	p.Stub, err = stub.New(p, opts...)

	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create plugin stub: %v\n", err)
		os.Exit(1)
	}

	p.Stub.UpdateContainers([]*api.ContainerUpdate{})

	err = p.Stub.Run(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "plugin exited with error: %v\n", err)
		os.Exit(1)
	}
}
