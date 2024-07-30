package nri

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/LionelJouin/network-dra/api/dra.networking/v1alpha1"
	"github.com/containerd/nri/pkg/api"
	"github.com/containerd/nri/pkg/stub"
	"github.com/containernetworking/cni/libcni"
	"github.com/containernetworking/cni/pkg/invoke"
	cnitypes "github.com/containernetworking/cni/pkg/types"
	netdefv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	resourcev1alpha3 "k8s.io/api/resource/v1alpha3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

type Plugin struct {
	Stub        stub.Stub
	ClientSet   clientset.Interface
	Exec        invoke.Exec
	CNIPath     []string
	CNICacheDir string
}

func (p *Plugin) RunPodSandbox(ctx context.Context, pod *api.PodSandbox) error {
	klog.FromContext(ctx).Info("RunPodSandbox", "pod.Name", pod.Name)

	resourceclaimList, err := p.ClientSet.ResourceV1alpha3().ResourceClaims(pod.Namespace).List(ctx, v1.ListOptions{})
	if err != nil {
		klog.FromContext(ctx).Error(err, "error getting ResourceClaims", "pod.Name", pod.Name)
		return fmt.Errorf("error getting ResourceClaims for pod '%s' in namespace '%s': %v", pod.Name, pod.Namespace, err)
	}

	podNetworkNamespace := getNetworkNamespace(pod)
	if podNetworkNamespace == "" {
		klog.FromContext(ctx).Error(err, "error getting network namespace", "pod.Name", pod.Name)
		return fmt.Errorf("error getting network namespace for pod '%s' in namespace '%s': %v", pod.Name, pod.Namespace, err)
	}

	for _, resourceClaim := range resourceclaimList.Items {
		if len(resourceClaim.Spec.Devices.Requests) != 1 && resourceClaim.Spec.Devices.Requests[0].DeviceClassName != v1alpha1.GroupName {
			continue
		}

		if len(resourceClaim.Status.ReservedFor) != 1 ||
			resourceClaim.Status.ReservedFor[0].Name != pod.GetName() ||
			resourceClaim.Status.ReservedFor[0].UID != types.UID(pod.GetUid()) {
			continue
		}

		if len(resourceClaim.Status.Allocation.Devices.Config) != 1 {
			continue
		}

		result, err := p.createAttachment(ctx, &resourceClaim, pod, podNetworkNamespace)
		if err != nil {
			klog.FromContext(ctx).Error(err, "error createAttachment", "pod.Name", pod.Name)
			return fmt.Errorf("error createAttachment for pod '%s' in namespace '%s': %v", pod.Name, pod.Namespace, err)
		}

		klog.FromContext(ctx).Info("createAttachment", "pod.Name", pod.Name, "resourceClaim.Name", resourceClaim.Name, "result", result)
	}

	return nil
}

func (p *Plugin) createAttachment(ctx context.Context, resourceClaim *resourcev1alpha3.ResourceClaim, pod *api.PodSandbox, podNetworkNamespace string) (cnitypes.Result, error) {
	networkAttachment := &v1alpha1.NetworkAttachment{}

	err := json.Unmarshal(resourceClaim.Status.Allocation.Devices.Config[0].Opaque.Parameters.Raw, networkAttachment)
	if err != nil {
		return nil, fmt.Errorf("failed to json.Unmarshal v1alpha1.NetworkAttachment: %v", err)
	}

	networkAttachmentDefinition := &netdefv1.NetworkAttachmentDefinition{}

	err = json.Unmarshal(networkAttachment.Status.NetworkRepresentation.Raw, networkAttachmentDefinition)
	if err != nil {
		return nil, fmt.Errorf("failed to json.Unmarshal netdefv1.NetworkAttachmentDefinition: %v", err)
	}

	cniNet := libcni.NewCNIConfigWithCacheDir(p.CNIPath, p.CNICacheDir, p.Exec)

	confList, err := libcni.ConfListFromBytes([]byte(networkAttachmentDefinition.Spec.Config))
	if err != nil {
		return nil, fmt.Errorf("failed to ConfListFromBytes: %v", err)
	}

	rt := &libcni.RuntimeConf{
		ContainerID: pod.GetId(),
		NetNS:       podNetworkNamespace,
		IfName:      networkAttachment.Spec.InterfaceRequest,
		Args: [][2]string{
			{"IgnoreUnknown", "true"},
			{"K8S_POD_NAMESPACE", pod.GetNamespace()},
			{"K8S_POD_NAME", pod.GetName()},
			{"K8S_POD_INFRA_CONTAINER_ID", pod.GetId()},
			{"K8S_POD_UID", pod.GetUid()},
		},
	}

	result, err := cniNet.AddNetworkList(ctx, confList, rt)
	if err != nil {
		return nil, fmt.Errorf("failed to AddNetwork: %v", err)
	}

	return result, nil
}

func getNetworkNamespace(pod *api.PodSandbox) string {
	for _, namespace := range pod.Linux.GetNamespaces() {
		if namespace.Type == "network" {
			return namespace.Path
		}
	}

	return ""
}
