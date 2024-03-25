package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/LionelJouin/network-dra/api/v1alpha1"
	"github.com/LionelJouin/network-dra/pkg/controllers"
	"github.com/spf13/cobra"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/dynamic-resource-allocation/controller"
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
	clientCfg, err := rest.InClusterConfig()
	if err != nil {
		fmt.Printf("failed to InClusterConfig: %v", err)
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(clientCfg)
	if err != nil {
		fmt.Printf("failed to NewForConfig: %v", err)
		os.Exit(1)
	}

	driverController := controllers.DriverController{}

	informerFactory := informers.NewSharedInformerFactory(clientset, 0)
	ctrl := controller.New(ctx, v1alpha1.GroupName, driverController, clientset, informerFactory)
	informerFactory.Start(ctx.Done())

	ctrl.Run(1)
}
