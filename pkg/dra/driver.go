package dra

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/LionelJouin/network-dra/api/dra.networking/v1alpha1"
	dranetworkingclientset "github.com/LionelJouin/network-dra/pkg/client/clientset/versioned"
	netdefv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	netdefclientset "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/client/clientset/versioned"
	resourcev1alpha2 "k8s.io/api/resource/v1alpha2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
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

	ClientSet              clientset.Interface
	NetDefClientSet        netdefclientset.Interface
	DRANetworkingClientSet dranetworkingclientset.Interface

	cdi *CDIHandler
}

func (d *Driver) Start(ctx context.Context) error {
	err := os.MkdirAll(d.DriverPluginPath, 0o750)
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
		claimSpec, err := d.getClaimParams(ctx, claim)
		if err != nil {
			return fmt.Errorf("error getting spec (NetworkAttachmentSpec + NetworkAttachmentDefinitionSpec) for claim %v: %v", claim.Uid, err)
		}

		err = d.cdi.CreateCDISpecFile(claim, claimSpec)
		if err != nil {
			return fmt.Errorf("error CreateCDISpecFile for claim %v: %v", claim.Uid, err)
		}

		prepared, err = d.cdi.GetClaimDevices(claim.Uid)
		if err != nil {
			return fmt.Errorf("error GetClaimDevices for claim %v: %v", claim.Uid, err)
		}

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

func (d *Driver) getClaimParams(ctx context.Context, claim *drapbv1.Claim) (*ClaimParams, error) {
	resourceClaim, err := d.getClaim(ctx, claim)
	if err != nil {
		return nil, err
	}

	networkAttachment, err := d.getNetworkAttachment(ctx, resourceClaim)
	if err != nil {
		return nil, err
	}

	networkAttachmentDefinition, err := d.getNetworkAttachmentDefinition(ctx, networkAttachment)
	if err != nil {
		return nil, err
	}

	return &ClaimParams{
		NetworkAttachmentSpec:           &networkAttachment.Spec,
		NetworkAttachmentDefinitionSpec: &networkAttachmentDefinition.Spec,
	}, nil
}

func (d *Driver) getClaim(ctx context.Context, claim *drapbv1.Claim) (*resourcev1alpha2.ResourceClaim, error) {
	resourceClaim, err := d.ClientSet.ResourceV1alpha2().ResourceClaims(claim.Namespace).Get(ctx, claim.Name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("error getting ResourceClaim '%v' in namespace '%v': %v", claim.Name, claim.Namespace, err)
	}

	return resourceClaim, nil
}

func (d *Driver) getNetworkAttachment(ctx context.Context, resourceClaim *resourcev1alpha2.ResourceClaim) (*v1alpha1.NetworkAttachment, error) {
	if resourceClaim.Spec.ParametersRef == nil {
		return nil, fmt.Errorf("ParametersRef cannot be nil in ResourceClaim %v in namespace %v", resourceClaim.Name, resourceClaim.Namespace)
	}

	networkAttachment, err := d.DRANetworkingClientSet.DraV1alpha1().NetworkAttachments(resourceClaim.Namespace).Get(ctx, resourceClaim.Spec.ParametersRef.Name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("error getting NetworkAttachment '%v' in namespace '%v': %v", resourceClaim.Spec.ParametersRef.Name, resourceClaim.Namespace, err)
	}

	return networkAttachment, nil
}

func (d *Driver) getNetworkAttachmentDefinition(ctx context.Context, networkAttachment *v1alpha1.NetworkAttachment) (*netdefv1.NetworkAttachmentDefinition, error) {
	networkAttachmentDefinition, err := d.NetDefClientSet.K8sCniCncfIoV1().NetworkAttachmentDefinitions(networkAttachment.Namespace).Get(ctx, networkAttachment.Spec.Name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("error getting NetworkAttachmentDefinition '%v' in namespace '%v': %v", networkAttachment.Spec.Name, networkAttachment.Namespace, err)
	}

	return networkAttachmentDefinition, nil
}
