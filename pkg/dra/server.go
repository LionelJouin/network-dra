package dra

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/LionelJouin/network-dra/pkg/oci/api/v1alpha1"
	"github.com/opencontainers/runtime-spec/specs-go"
	runtimespec "github.com/opencontainers/runtime-spec/specs-go"
	cri "k8s.io/cri-api/pkg/apis/runtime/v1"
	"k8s.io/klog/v2"
)

type OCIHookCallbackServer struct {
	v1alpha1.UnimplementedOCIHookServer

	Client cri.RuntimeServiceClient
}

type PodSandboxStatusInfo struct {
	RuntimeSpec *runtimespec.Spec `json:"runtimeSpec"`
}

func (ocihcs *OCIHookCallbackServer) CreateContainer(ctx context.Context, createContainerRequest *v1alpha1.CreateContainerRequest) (*v1alpha1.CreateContainerResponse, error) {
	klog.FromContext(ctx).Info("CreateContainer", "Claim", createContainerRequest.Claim, "OciState", createContainerRequest.OciState)

	var ociState runtimespec.State
	err := json.Unmarshal([]byte(createContainerRequest.OciState), &ociState)
	if err != nil {
		return nil, fmt.Errorf("failed to json.Unmarshal ociState: %w", err)
	}

	podSandboxId, exists := ociState.Annotations["io.kubernetes.cri.sandbox-id"]
	if !exists {
		klog.FromContext(ctx).Error(nil, "io.kubernetes.cri.sandbox-id doesn't exist in ociState")
		return nil, fmt.Errorf("io.kubernetes.cri.sandbox-id doesn't exist in ociState")
	}

	podSandboxStatus, err := ocihcs.Client.PodSandboxStatus(ctx, &cri.PodSandboxStatusRequest{
		PodSandboxId: podSandboxId,
		Verbose:      true,
	})
	if err != nil || podSandboxStatus == nil {
		klog.FromContext(ctx).Error(err, "failed to PodSandboxStatus for PodSandboxId", podSandboxId)
		return nil, fmt.Errorf("failed to PodSandboxStatus for PodSandboxId %s: %w", podSandboxId, err)
	}

	sandboxInfo := &PodSandboxStatusInfo{}

	if err := json.Unmarshal([]byte(podSandboxStatus.Info["info"]), sandboxInfo); err != nil {
		klog.FromContext(ctx).Error(err, "failed to Unmarshal podSandboxStatus.Info['info']")
		return nil, fmt.Errorf("failed to Unmarshal podSandboxStatus.Info['info']: %w", err)
	}

	networkNamespace := ""

	for _, namespace := range sandboxInfo.RuntimeSpec.Linux.Namespaces {
		if namespace.Type != specs.NetworkNamespace {
			continue
		}

		networkNamespace = namespace.Path
		break
	}

	if networkNamespace == "" {
		klog.FromContext(ctx).Error(err, "failed to network namespace for PodSandboxId", podSandboxId)
		return nil, fmt.Errorf("failed to find network namespace for PodSandboxId %s: %w", podSandboxId, err)
	}

	klog.FromContext(ctx).Info("CreateContainer", "network namespace", networkNamespace)

	return &v1alpha1.CreateContainerResponse{}, nil
}
