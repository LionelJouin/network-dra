package dra

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/LionelJouin/network-dra/pkg/oci/api/v1alpha1"
	"github.com/k8snetworkplumbingwg/multus-dynamic-networks-controller/pkg/multuscni"
	nadutils "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/utils"
	runtimespec "github.com/opencontainers/runtime-spec/specs-go"
	multusapi "gopkg.in/k8snetworkplumbingwg/multus-cni.v4/pkg/server/api"
	v1alpha2resource "k8s.io/api/resource/v1alpha2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	cri "k8s.io/cri-api/pkg/apis/runtime/v1"
	"k8s.io/klog/v2"
)

type OCIHookCallbackServer struct {
	v1alpha1.UnimplementedOCIHookServer

	Name         string
	CRIClient    cri.RuntimeServiceClient
	MultusClient multuscni.Client
	ClientSet    clientset.Interface
}

type PodSandboxStatusInfo struct {
	RuntimeSpec *runtimespec.Spec `json:"runtimeSpec"`
}

func (ocihcs *OCIHookCallbackServer) CreateRuntime(ctx context.Context, createRuntimeRequest *v1alpha1.CreateRuntimeRequest) (*v1alpha1.CreateRuntimeResponse, error) {
	klog.FromContext(ctx).Info("CreateRuntime", "OciState", createRuntimeRequest.OciState, "ClaimUID", createRuntimeRequest.ClaimUID, "ClaimSpec", createRuntimeRequest.ClaimSpec)

	claimSpec := &ClaimParams{}
	err := claimSpec.decode(createRuntimeRequest.ClaimSpec)
	if err != nil {
		klog.FromContext(ctx).Error(err, "failed to decode ClaimSpec")
		return nil, err
	}

	if claimSpec.NetworkAttachmentSpec == nil || claimSpec.NetworkAttachmentDefinitionSpec == nil {
		klog.FromContext(ctx).Error(nil, "NetworkAttachmentSpec and NetworkAttachmentDefinitionSpec cannot be nil")
		return nil, err
	}

	podSandboxId, podName, podNamespace, podUId, err := getPodInfos(ctx, createRuntimeRequest.OciState)
	if err != nil {
		return nil, err
	}

	networkNamespace, err := ocihcs.getNetworkNamespace(ctx, podSandboxId)
	if err != nil {
		return nil, err
	}

	netAttachDefWithDefaults, err := nadutils.GetCNIConfigFromSpec(claimSpec.NetworkAttachmentDefinitionSpec.Config, claimSpec.NetworkAttachmentSpec.Name)
	if err != nil {
		klog.FromContext(ctx).Error(err, "failed to GetCNIConfigFromSpec")
		return nil, err
	}

	klog.FromContext(ctx).Info("CreateRuntime", "network namespace", networkNamespace)

	_, err = ocihcs.MultusClient.InvokeDelegate(multusapi.CreateDelegateRequest(
		multuscni.CmdAdd,
		podSandboxId,
		networkNamespace,
		claimSpec.NetworkAttachmentSpec.InterfaceRequest,
		podNamespace,
		podName,
		podUId,
		netAttachDefWithDefaults,
		&multusapi.DelegateInterfaceAttributes{
			IPRequest:  claimSpec.NetworkAttachmentSpec.IPRequest,
			MacRequest: claimSpec.NetworkAttachmentSpec.MacRequest,
		},
	))
	if err != nil {
		klog.FromContext(ctx).Error(err, "failed to multusClient.InvokeDelegate")
		return nil, err
	}

	// status.allocation: Invalid value: ... field is immutable
	// err = ocihcs.updateStatus(ctx, createRuntimeRequest, claimSpec, response)
	// if err != nil {
	// 	klog.FromContext(ctx).Error(err, "failed to updateStatus")
	// 	return nil, err
	// }

	return &v1alpha1.CreateRuntimeResponse{}, nil
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

func (ocihcs *OCIHookCallbackServer) updateStatus(ctx context.Context, createRuntimeRequest *v1alpha1.CreateRuntimeRequest, claimSpec *ClaimParams, response *multusapi.Response) error {
	resourceClaim, err := ocihcs.ClientSet.ResourceV1alpha2().ResourceClaims(createRuntimeRequest.ClaimNamespace).Get(ctx, createRuntimeRequest.ClaimName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting ResourceClaim '%v' in namespace '%v': %v", createRuntimeRequest.ClaimName, createRuntimeRequest.ClaimNamespace, err)
	}

	status, err := nadutils.CreateNetworkStatus(
		response.Result,
		fmt.Sprintf("%s/%s", claimSpec.NetworkAttachmentSpec.Namespace, claimSpec.NetworkAttachmentSpec.Name),
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to create NetworkStatus from the response: %v", err)
	}

	statusStr, err := json.Marshal(status)
	if err != nil {
		return fmt.Errorf("failed to json.Marshal status: %v", err)
	}

	resourceClaim.Status.Allocation.ResourceHandles = []v1alpha2resource.ResourceHandle{
		{
			DriverName: ocihcs.Name,
			Data:       string(statusStr),
		},
	}

	_, err = ocihcs.ClientSet.ResourceV1alpha2().ResourceClaims(createRuntimeRequest.ClaimNamespace).UpdateStatus(ctx, resourceClaim, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("error updating status of ResourceClaim '%v' in namespace '%v': %v", createRuntimeRequest.ClaimName, createRuntimeRequest.ClaimNamespace, err)
	}

	return nil
}
