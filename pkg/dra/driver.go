package dra

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	plugin "k8s.io/dynamic-resource-allocation/kubeletplugin"
	"k8s.io/klog/v2"
	drapbv1 "k8s.io/kubelet/pkg/apis/dra/v1alpha4"
)

type Driver struct {
	Name                   string
	DriverPluginPath       string
	PluginRegistrationPath string
	NodeName               string
	Clientset              kubernetes.Interface
}

func (d *Driver) Start(ctx context.Context) error {
	err := os.MkdirAll(d.DriverPluginPath, 0750)
	if err != nil {
		return fmt.Errorf("failed to MkdirAll DriverPluginPath %v: %w", d.DriverPluginPath, err)
	}

	driverPluginPath := filepath.Join(d.DriverPluginPath, "plugin.sock")

	dp, err := plugin.Start(
		ctx,
		d,
		plugin.DriverName(d.Name),
		plugin.NodeName(d.NodeName),
		plugin.KubeClient(d.Clientset),
		plugin.RegistrarSocketPath(d.PluginRegistrationPath),
		plugin.PluginSocketPath(driverPluginPath),
		plugin.KubeletPluginSocketPath(driverPluginPath))
	if err != nil {
		return fmt.Errorf("failed to start DRA driver plugin: %w", err)
	}

	klog.FromContext(ctx).Info("Driver/Plugin Started", "Name", d.Name, "DriverPluginPath", d.DriverPluginPath, "PluginRegistrationPath", d.PluginRegistrationPath)

	<-ctx.Done()

	dp.Stop()

	return nil
}

func (d *Driver) NodePrepareResources(ctx context.Context, req *drapbv1.NodePrepareResourcesRequest) (*drapbv1.NodePrepareResourcesResponse, error) {
	klog.FromContext(ctx).Info("NodePrepareResource", "numClaims", len(req.Claims))
	preparedResources := &drapbv1.NodePrepareResourcesResponse{Claims: map[string]*drapbv1.NodePrepareResourceResponse{}}

	// In production version some common operations of d.nodeUnprepareResources
	// should be done outside of the loop, for instance updating the CR could
	// be done once after all HW was prepared.
	for _, claim := range req.Claims {
		preparedResources.Claims[claim.UID] = &drapbv1.NodePrepareResourceResponse{
			Devices: []*drapbv1.Device{
				{
					PoolName:   "none",
					DeviceName: "none",
				},
			},
		}
	}

	return preparedResources, nil
}

func (d *Driver) NodeUnprepareResources(ctx context.Context, req *drapbv1.NodeUnprepareResourcesRequest) (*drapbv1.NodeUnprepareResourcesResponse, error) {
	klog.FromContext(ctx).Info("NodeUnprepareResource", "numClaims", len(req.Claims))
	unpreparedResources := &drapbv1.NodeUnprepareResourcesResponse{}

	return unpreparedResources, nil
}
