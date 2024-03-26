package dra

import (
	"context"

	"github.com/LionelJouin/network-dra/pkg/oci/api/v1alpha1"
	"k8s.io/klog/v2"
)

type OCIHookCallbackServer struct {
	v1alpha1.UnimplementedOCIHookServer
}

func (ocihcs *OCIHookCallbackServer) CreateContainer(ctx context.Context, createContainerRequest *v1alpha1.CreateContainerRequest) (*v1alpha1.CreateContainerResponse, error) {
	klog.FromContext(ctx).Info("CreateContainer", "Claim", createContainerRequest.Claim, "OciState", createContainerRequest.OciState)
	return &v1alpha1.CreateContainerResponse{}, nil
}
