package dra

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/LionelJouin/network-dra/pkg/oci/api/v1alpha1"
	"github.com/k8snetworkplumbingwg/multus-dynamic-networks-controller/pkg/multuscni"
	runtimespec "github.com/opencontainers/runtime-spec/specs-go"
	multusapi "gopkg.in/k8snetworkplumbingwg/multus-cni.v4/pkg/server/api"
	cri "k8s.io/cri-api/pkg/apis/runtime/v1"
	"k8s.io/klog/v2"
)

type OCIHookCallbackServer struct {
	v1alpha1.UnimplementedOCIHookServer

	CRIClient    cri.RuntimeServiceClient
	MultusClient multuscni.Client
}

type PodSandboxStatusInfo struct {
	RuntimeSpec *runtimespec.Spec `json:"runtimeSpec"`
}

func (ocihcs *OCIHookCallbackServer) CreateContainer(ctx context.Context, createContainerRequest *v1alpha1.CreateContainerRequest) (*v1alpha1.CreateContainerResponse, error) {
	klog.FromContext(ctx).Info("CreateContainer", "Claim", createContainerRequest.Claim, "OciState", createContainerRequest.OciState)

	podSandboxId, podName, podNamespace, podUId, err := getPodInfos(ctx, createContainerRequest.OciState)
	if err != nil {
		return nil, err
	}

	networkNamespace, err := ocihcs.getNetworkNamespace(ctx, podSandboxId)
	if err != nil {
		return nil, err
	}

	cniConfig := `{
		"name": "mynet",
		"type": "macvlan",
		"master": "eth0",
		"linkInContainer": false
	}`

	klog.FromContext(ctx).Info("CreateContainer", "network namespace", networkNamespace)

	_, err = ocihcs.MultusClient.InvokeDelegate(multusapi.CreateDelegateRequest(
		multuscni.CmdAdd,
		podSandboxId,
		networkNamespace,
		"net-1",
		podNamespace,
		podName,
		podUId,
		[]byte(cniConfig),
		&multusapi.DelegateInterfaceAttributes{},
	))
	if err != nil {
		klog.FromContext(ctx).Error(nil, "failed to multusClient.InvokeDelegate")
		return nil, err
	}

	return &v1alpha1.CreateContainerResponse{}, nil
}

// returns podSandboxId, pod name, pod namespace, pod uid
func getPodInfos(ctx context.Context, OciStateStr string) (string, string, string, string, error) {
	var ociState runtimespec.State
	err := json.Unmarshal([]byte(OciStateStr), &ociState)
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to json.Unmarshal ociState: %w", err)
	}

	podSandboxId, exists := ociState.Annotations["io.kubernetes.cri.sandbox-id"]
	if !exists {
		klog.FromContext(ctx).Error(nil, "io.kubernetes.cri.sandbox-id doesn't exist in ociState")
		return "", "", "", "", fmt.Errorf("io.kubernetes.cri.sandbox-id doesn't exist in ociState")
	}

	podName, exists := ociState.Annotations["io.kubernetes.cri.sandbox-name"]
	if !exists {
		klog.FromContext(ctx).Error(nil, "io.kubernetes.cri.sandbox-name doesn't exist in ociState")
		return "", "", "", "", fmt.Errorf("io.kubernetes.cri.sandbox-name doesn't exist in ociState")
	}

	podNamespace, exists := ociState.Annotations["io.kubernetes.cri.sandbox-namespace"]
	if !exists {
		klog.FromContext(ctx).Error(nil, "io.kubernetes.cri.sandbox-namespace doesn't exist in ociState")
		return "", "", "", "", fmt.Errorf("io.kubernetes.cri.sandbox-namespace doesn't exist in ociState")
	}

	podUId, exists := ociState.Annotations["io.kubernetes.cri.sandbox-uid"]
	if !exists {
		klog.FromContext(ctx).Error(nil, "io.kubernetes.cri.sandbox-uid doesn't exist in ociState")
		return "", "", "", "", fmt.Errorf("io.kubernetes.cri.sandbox-uid doesn't exist in ociState")
	}

	return podSandboxId, podName, podNamespace, podUId, nil
}

func (ocihcs *OCIHookCallbackServer) getNetworkNamespace(ctx context.Context, podSandboxId string) (string, error) {
	podSandboxStatus, err := ocihcs.CRIClient.PodSandboxStatus(ctx, &cri.PodSandboxStatusRequest{
		PodSandboxId: podSandboxId,
		Verbose:      true,
	})
	if err != nil || podSandboxStatus == nil {
		klog.FromContext(ctx).Error(err, "failed to PodSandboxStatus for PodSandboxId", podSandboxId)
		return "", fmt.Errorf("failed to PodSandboxStatus for PodSandboxId %s: %w", podSandboxId, err)
	}

	sandboxInfo := &PodSandboxStatusInfo{}

	if err := json.Unmarshal([]byte(podSandboxStatus.Info["info"]), sandboxInfo); err != nil {
		klog.FromContext(ctx).Error(err, "failed to Unmarshal podSandboxStatus.Info['info']")
		return "", fmt.Errorf("failed to Unmarshal podSandboxStatus.Info['info']: %w", err)
	}

	networkNamespace := ""

	for _, namespace := range sandboxInfo.RuntimeSpec.Linux.Namespaces {
		if namespace.Type != runtimespec.NetworkNamespace {
			continue
		}

		networkNamespace = namespace.Path
		break
	}

	if networkNamespace == "" {
		klog.FromContext(ctx).Error(err, "failed to network namespace for PodSandboxId", podSandboxId)
		return "", fmt.Errorf("failed to find network namespace for PodSandboxId %s: %w", podSandboxId, err)
	}

	return networkNamespace, nil
}
