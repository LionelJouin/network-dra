package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/LionelJouin/network-dra/api/dra.networking/v1alpha1"
	dranetworkingclientset "github.com/LionelJouin/network-dra/pkg/client/clientset/versioned"
	netdefv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	netdefclientset "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/client/clientset/versioned"
	v1 "k8s.io/api/core/v1"
	resourcev1alpha2 "k8s.io/api/resource/v1alpha2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/dynamic-resource-allocation/controller"
)

type DriverController struct {
	NetDefClientSet        netdefclientset.Interface
	DRANetworkingClientSet dranetworkingclientset.Interface
}

func (dc DriverController) GetClassParameters(ctx context.Context, class *resourcev1alpha2.ResourceClass) (interface{}, error) {
	return nil, nil
}

func (dc DriverController) GetClaimParameters(ctx context.Context, claim *resourcev1alpha2.ResourceClaim, class *resourcev1alpha2.ResourceClass, classParameters interface{}) (interface{}, error) {
	networkAttachment, err := dc.getNetworkAttachment(ctx, claim)
	if err != nil {
		return nil, err
	}

	networkAttachmentDefinition, err := dc.getNetworkAttachmentDefinition(ctx, networkAttachment)
	if err != nil {
		return nil, err
	}

	return &v1alpha1.ClaimParameter{
		NetworkAttachment:           networkAttachment,
		NetworkAttachmentDefinition: networkAttachmentDefinition,
	}, nil
}

func (dc DriverController) Allocate(ctx context.Context, claims []*controller.ClaimAllocation, selectedNode string) {
	// In production version of the driver the common operations for every
	// d.allocate looped call should be done prior this loop, and can be reused
	// for every d.allocate() looped call.
	// E.g.: selectedNode=="" check, client stup and CRD fetching.
	for _, ca := range claims {
		ca.Allocation, ca.Error = dc.allocate(ctx, ca.Claim, ca.ClaimParameters, ca.Class, ca.ClassParameters, selectedNode)
	}
}

func (dc DriverController) Deallocate(ctx context.Context, claim *resourcev1alpha2.ResourceClaim) error {
	return nil
}

func (dc DriverController) UnsuitableNodes(ctx context.Context, pod *v1.Pod, claims []*controller.ClaimAllocation, potentialNodes []string) error {
	return nil
}

func (dc *DriverController) allocate(ctx context.Context, claim *resourcev1alpha2.ResourceClaim, claimParameters interface{}, class *resourcev1alpha2.ResourceClass, classParameters interface{}, selectedNode string) (*resourcev1alpha2.AllocationResult, error) {
	params, ok := claimParameters.(*v1alpha1.ClaimParameter)
	if !ok {
		return nil, errors.New("claimParameters must be type *v1alpha1.ClaimParameter")
	}

	paramsStr, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to json.Marshal params: %v", err)
	}

	allocationResult := &resourcev1alpha2.AllocationResult{
		ResourceHandles: []resourcev1alpha2.ResourceHandle{
			{
				DriverName: v1alpha1.GroupName,
				Data:       string(paramsStr),
			},
		},
		AvailableOnNodes: &v1.NodeSelector{
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
		Shareable: false,
	}
	return allocationResult, nil
}

func (dc *DriverController) getNetworkAttachment(ctx context.Context, resourceClaim *resourcev1alpha2.ResourceClaim) (*v1alpha1.NetworkAttachment, error) {
	if resourceClaim.Spec.ParametersRef == nil {
		return nil, fmt.Errorf("ParametersRef cannot be nil in ResourceClaim %v in namespace %v", resourceClaim.Name, resourceClaim.Namespace)
	}

	networkAttachment, err := dc.DRANetworkingClientSet.DraV1alpha1().NetworkAttachments(resourceClaim.Namespace).Get(ctx, resourceClaim.Spec.ParametersRef.Name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("error getting NetworkAttachment '%v' in namespace '%v': %v", resourceClaim.Spec.ParametersRef.Name, resourceClaim.Namespace, err)
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
