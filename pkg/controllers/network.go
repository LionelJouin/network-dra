package controllers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/LionelJouin/network-dra/api/dra.networking/v1alpha1"
	dranetworkingclientset "github.com/LionelJouin/network-dra/pkg/client/clientset/versioned"
	netdefv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	netdefclientset "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/client/clientset/versioned"
	v1 "k8s.io/api/core/v1"
	resourcev1alpha3 "k8s.io/api/resource/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/dynamic-resource-allocation/controller"
)

type DriverController struct {
	NetDefClientSet        netdefclientset.Interface
	DRANetworkingClientSet dranetworkingclientset.Interface
}

func (dc DriverController) Allocate(ctx context.Context, claims []*controller.ClaimAllocation, selectedNode string) {
	// In production version of the driver the common operations for every
	// d.allocate looped call should be done prior this loop, and can be reused
	// for every d.allocate() looped call.
	// E.g.: selectedNode=="" check, client stup and CRD fetching.
	for _, ca := range claims {
		ca.Allocation, ca.Error = dc.allocate(ctx, ca.Claim, selectedNode)
	}
}

func (dc DriverController) Deallocate(ctx context.Context, claim *resourcev1alpha3.ResourceClaim) error {
	return nil
}

func (dc DriverController) UnsuitableNodes(ctx context.Context, pod *v1.Pod, claims []*controller.ClaimAllocation, potentialNodes []string) error {
	return nil
}

func (dc *DriverController) allocate(ctx context.Context, claim *resourcev1alpha3.ResourceClaim, selectedNode string) (*resourcev1alpha3.AllocationResult, error) {
	networkAttachment, err := dc.getNetworkAttachment(ctx, claim)
	if err != nil {
		return nil, err
	}

	networkAttachmentDefinition, err := dc.getNetworkAttachmentDefinition(ctx, networkAttachment)
	if err != nil {
		return nil, err
	}

	networkAttachment.Status.NetworkRepresentation = runtime.RawExtension{
		Object: networkAttachmentDefinition,
	}

	allocationResult := &resourcev1alpha3.AllocationResult{
		Devices: resourcev1alpha3.DeviceAllocationResult{
			Config: []resourcev1alpha3.DeviceAllocationConfiguration{
				{
					Source: resourcev1alpha3.AllocationConfigSourceClaim,
					DeviceConfiguration: resourcev1alpha3.DeviceConfiguration{
						Opaque: &resourcev1alpha3.OpaqueDeviceConfiguration{
							Driver: v1alpha1.GroupName,
							Parameters: runtime.RawExtension{
								Object: networkAttachment,
							},
						},
					},
				},
			},
		},
		NodeSelector: &v1.NodeSelector{
			NodeSelectorTerms: []v1.NodeSelectorTerm{
				{
					MatchFields: []v1.NodeSelectorRequirement{
						{
							Key:      "metadata.name",
							Operator: "In",
							Values:   []string{selectedNode},
						},
					},
				},
			},
		},
	}
	return allocationResult, nil
}

func (dc *DriverController) getNetworkAttachment(ctx context.Context, resourceClaim *resourcev1alpha3.ResourceClaim) (*v1alpha1.NetworkAttachment, error) {
	networkAttachment := &v1alpha1.NetworkAttachment{}

	err := json.Unmarshal(resourceClaim.Spec.Devices.Config[0].Opaque.Parameters.Raw, networkAttachment)
	if err != nil {
		return nil, fmt.Errorf("failed to json.Unmarshal v1alpha1.NetworkAttachment: %v", err)
	}

	return networkAttachment, nil
}

func (dc *DriverController) getNetworkAttachmentDefinition(ctx context.Context, networkAttachment *v1alpha1.NetworkAttachment) (*netdefv1.NetworkAttachmentDefinition, error) {
	networkAttachmentDefinition, err := dc.NetDefClientSet.K8sCniCncfIoV1().NetworkAttachmentDefinitions(networkAttachment.Namespace).Get(ctx, networkAttachment.Spec.Name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("error getting NetworkAttachmentDefinition '%v' in namespace '%v': %v", networkAttachment.Spec.Name, networkAttachment.Namespace, err)
	}

	return networkAttachmentDefinition, nil
}
