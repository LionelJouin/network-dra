package dra

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/util/retry"
	plugin "k8s.io/dynamic-resource-allocation/kubeletplugin"
	"k8s.io/klog/v2"
	drapbv1 "k8s.io/kubelet/pkg/apis/dra/v1alpha3"
)

type Driver struct {
	Name                   string
	DriverPluginPath       string
	PluginRegistrationPath string
	CDIRoot                string
	OCIHookPath            string
	OCIHookSocketPath      string

	cdi *CDIHandler
}

func (d *Driver) Start(ctx context.Context) error {
	err := os.MkdirAll(d.DriverPluginPath, 0750)
	if err != nil {
		return fmt.Errorf("failed to MkdirAll DriverPluginPath %v: %w", d.DriverPluginPath, err)
	}

	d.cdi, err = NewCDIHandler(d.CDIRoot, d.OCIHookPath, d.OCIHookSocketPath)
	if err != nil {
		return fmt.Errorf("failed to create cdi handler: %w", err)
	}

	driverPluginPath := filepath.Join(d.DriverPluginPath, "plugin.sock")

	dp, err := plugin.Start(
		d,
		plugin.DriverName(d.Name),
		plugin.RegistrarSocketPath(d.PluginRegistrationPath),
		plugin.PluginSocketPath(driverPluginPath),
		plugin.KubeletPluginSocketPath(driverPluginPath))
	if err != nil {
		return fmt.Errorf("failed to start DRA driver plugin: %w", err)
	}

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
		preparedResources.Claims[claim.Uid] = d.nodePrepareResource(ctx, claim)
	}

	return preparedResources, nil
}

func (d *Driver) NodeUnprepareResources(ctx context.Context, req *drapbv1.NodeUnprepareResourcesRequest) (*drapbv1.NodeUnprepareResourcesResponse, error) {
	klog.FromContext(ctx).Info("NodeUnprepareResource", "numClaims", len(req.Claims))
	unpreparedResources := &drapbv1.NodeUnprepareResourcesResponse{
		Claims: map[string]*drapbv1.NodeUnprepareResourceResponse{},
	}

	// In production version some common operations of d.nodeUnprepareResources
	// should be done outside of the loop, for instance updating the CR could
	// be done once after all HW was unprepared.
	for _, claim := range req.Claims {
		unpreparedResources.Claims[claim.Uid] = d.nodeUnprepareResource(ctx, claim)
	}

	return unpreparedResources, nil
}

func (d *Driver) nodePrepareResource(ctx context.Context, claim *drapbv1.Claim) *drapbv1.NodePrepareResourceResponse {
	var err error
	var prepared []string
	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		err = d.cdi.CreateCDISpecFile(claim.Uid)
		if err != nil {
			return fmt.Errorf("error CreateCDISpecFile for claim %v: %v", claim.Uid, err)
		}

		prepared, err = d.cdi.GetClaimDevices(claim.Uid)
		if err != nil {
			return fmt.Errorf("error GetClaimDevices for claim %v: %v", claim.Uid, err)
		}

		// prepared, err = d.prepare(ctx, claim.Uid)
		// if err != nil {
		// 	return fmt.Errorf("error allocating devices for claim '%v': %v", claim.Uid, err)
		// }

		// updatedSpec, err := d.state.GetUpdatedSpec(&d.nascrd.Spec)
		// if err != nil {
		// 	return fmt.Errorf("error getting updated CR spec: %v", err)
		// }

		// err = d.nasclient.Update(ctx, updatedSpec)
		// if err != nil {
		// 	if err := d.state.Unprepare(claim.Uid); err != nil {
		// 		klog.FromContext(ctx).Error(err, "Failed to unprepare after Update", "claim", claim.Uid)
		// 	}
		// 	return err
		// }

		return nil
	})

	if err != nil {
		return &drapbv1.NodePrepareResourceResponse{
			Error: fmt.Sprintf("error preparing resource: %v", err),
		}
	}

	klog.FromContext(ctx).Info("Prepared devices", "claim", claim.Uid, "Name", claim.Name)
	return &drapbv1.NodePrepareResourceResponse{CDIDevices: prepared}
}

func (d *Driver) nodeUnprepareResource(ctx context.Context, claim *drapbv1.Claim) *drapbv1.NodeUnprepareResourceResponse {
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// err := d.unprepare(ctx, claim.Uid)
		// if err != nil {
		// 	return fmt.Errorf("error unpreparing devices for claim '%v': %v", claim.Uid, err)
		// }

		// updatedSpec, err := d.state.GetUpdatedSpec(&d.nascrd.Spec)
		// if err != nil {
		// 	return fmt.Errorf("error getting updated CR spec: %v", err)
		// }

		// err = d.nasclient.Update(ctx, updatedSpec)
		// if err != nil {
		// 	return err
		// }

		return nil
	})
	if err != nil {
		return &drapbv1.NodeUnprepareResourceResponse{
			Error: fmt.Sprintf("error unpreparing resource: %v", err),
		}
	}

	klog.FromContext(ctx).Info("Unprepared devices", "claim", claim.Uid, "Name", claim.Name)
	return &drapbv1.NodeUnprepareResourceResponse{}
}
