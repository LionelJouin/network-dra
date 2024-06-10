package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/LionelJouin/network-dra/api/dra.networking/v1alpha1"
	dranetworkingclientset "github.com/LionelJouin/network-dra/pkg/client/clientset/versioned"
	"github.com/LionelJouin/network-dra/pkg/controllers"
	netdefclientset "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/client/clientset/versioned"
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
		fmt.Fprintf(os.Stderr, "failed to InClusterConfig: %v\n", err)
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(clientCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to NewForConfig: %v\n", err)
		os.Exit(1)
	}

	netDefClientSet, err := netdefclientset.NewForConfig(clientCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to NewForConfig: %v\n", err)
		os.Exit(1)
	}

	draNetworkingClientSet, err := dranetworkingclientset.NewForConfig(clientCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to NewForConfig: %v\n", err)
		os.Exit(1)
	}

	driverController := controllers.DriverController{
		NetDefClientSet:        netDefClientSet,
		DRANetworkingClientSet: draNetworkingClientSet,
	}

	informerFactory := informers.NewSharedInformerFactory(clientset, 0)
	ctrl := controller.New(ctx, v1alpha1.GroupName, driverController, clientset, informerFactory)
	informerFactory.Start(ctx.Done())

	ctrl.Run(1)
}
