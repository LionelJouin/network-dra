package controllers

import (
	"context"

	v1 "k8s.io/api/core/v1"
	resourcev1alpha2 "k8s.io/api/resource/v1alpha2"
	"k8s.io/dynamic-resource-allocation/controller"
)

type DriverController struct{}

func (dc DriverController) GetClassParameters(ctx context.Context, class *resourcev1alpha2.ResourceClass) (interface{}, error) {
	return nil, nil
}

func (dc DriverController) GetClaimParameters(ctx context.Context, claim *resourcev1alpha2.ResourceClaim, class *resourcev1alpha2.ResourceClass, classParameters interface{}) (interface{}, error) {
	return nil, nil
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
	allocationResult := &resourcev1alpha2.AllocationResult{
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
