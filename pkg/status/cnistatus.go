package status

import (
	"context"
	"encoding/json"
	"fmt"

	cnitypes "github.com/containernetworking/cni/pkg/types"
	cni100 "github.com/containernetworking/cni/pkg/types/100"
	resourcev1alpha3 "k8s.io/api/resource/v1alpha3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientset "k8s.io/client-go/kubernetes"
)

type CNIStatusHandler struct {
	ClientSet clientset.Interface
}

func (cnish *CNIStatusHandler) UpdateStatus(ctx context.Context, claim *resourcev1alpha3.ResourceClaim, result cnitypes.Result) error {
	resultBytes, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("cni.handleClaim: failed to json.Marshal result (%v): %v", result, err)
	}

	cniResult, err := cni100.NewResultFromResult(result)
	if err != nil {
		return fmt.Errorf("cni.handleClaim: failed to NewResultFromResult result (%v): %v", result, err)
	}

	claim.Status.Devices = append(claim.Status.Devices, resourcev1alpha3.AllocatedDeviceStatus{
		Driver: claim.Status.Allocation.Devices.Results[0].Driver,
		Pool:   claim.Status.Allocation.Devices.Results[0].Pool,
		Device: claim.Status.Allocation.Devices.Results[0].Device,
		Data: []runtime.RawExtension{
			{
				Raw: resultBytes,
			},
		},
		NetworkData: cniResultToNetworkData(cniResult),
	})

	_, err = cnish.ClientSet.ResourceV1alpha3().ResourceClaims(claim.GetNamespace()).UpdateStatus(ctx, claim, v1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("cni.handleClaim: failed to update resource claim status (%v): %v", result, err)
	}
	return nil
}

func cniResultToNetworkData(cniResult *cni100.Result) resourcev1alpha3.NetworkDeviceData {
	networkData := resourcev1alpha3.NetworkDeviceData{}

	for _, ip := range cniResult.IPs {
		networkData.Addresses = append(networkData.Addresses, resourcev1alpha3.NetworkAddress{
			CIDR: ip.Address.String(),
		})
	}

	for _, ifs := range cniResult.Interfaces {
		// Only pod interfaces can have sandbox information
		if ifs.Sandbox != "" {
			networkData.InterfaceName = ifs.Name
			networkData.HWAddress = ifs.Mac
		}
	}

	return networkData
}
