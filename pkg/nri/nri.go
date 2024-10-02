package nri

import (
	"context"
	"fmt"

	"github.com/containerd/nri/pkg/api"
	"github.com/containerd/nri/pkg/stub"
	cniv1 "github.com/kubernetes-sigs/multi-network/pkg/cni/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

type Plugin struct {
	Stub      stub.Stub
	ClientSet clientset.Interface
	CNI       *cniv1.CNI
}

func (p *Plugin) RunPodSandbox(ctx context.Context, pod *api.PodSandbox) error {
	klog.FromContext(ctx).Info("RunPodSandbox", "pod.Name", pod.Name)

	podNetworkNamespace := getNetworkNamespace(pod)
	if podNetworkNamespace == "" {
		return fmt.Errorf("error getting network namespace for pod '%s' in namespace '%s'", pod.Name, pod.Namespace)
	}

	err := p.CNI.AttachNetworks(ctx, pod.Id, pod.Uid, pod.Name, pod.Namespace, podNetworkNamespace)
	if err != nil {
		return fmt.Errorf("error CNI.AttachNetworks for pod '%s' (uid: %s) in namespace '%s': %v", pod.Name, pod.Uid, pod.Namespace, err)
	}

	return nil
}

func getNetworkNamespace(pod *api.PodSandbox) string {
	for _, namespace := range pod.Linux.GetNamespaces() {
		if namespace.Type == "network" {
			return namespace.Path
		}
	}

	return ""
}
