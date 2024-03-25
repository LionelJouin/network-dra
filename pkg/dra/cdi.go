package dra

import (
	"fmt"
	"os"

	"github.com/LionelJouin/network-dra/api/v1alpha1"
	cdiapi "github.com/container-orchestrated-devices/container-device-interface/pkg/cdi"
	cdispec "github.com/container-orchestrated-devices/container-device-interface/specs-go"
	"tags.cncf.io/container-device-interface/pkg/parser"
)

const (
	cdiVendor = "k8s." + v1alpha1.GroupName
	cdiClass  = "attachement"
	cdiKind   = cdiVendor + "/" + cdiClass
)

type CDIHandler struct {
	registry cdiapi.Registry
}

func NewCDIHandler(cdiRoot string) (*CDIHandler, error) {
	info, err := os.Stat(cdiRoot)
	switch {
	case err != nil && os.IsNotExist(err):
		err := os.MkdirAll(cdiRoot, 0750)
		if err != nil {
			return nil, fmt.Errorf("failed to MkdirAll CDIRoot %v: %w", cdiRoot, err)
		}
	case err != nil:
		return nil, err
	case !info.IsDir():
		return nil, fmt.Errorf("path for cdi file generation is not a directory: %w", err)
	}

	registry := cdiapi.GetRegistry(
		cdiapi.WithSpecDirs(cdiRoot),
	)

	err = registry.Refresh()
	if err != nil {
		return nil, fmt.Errorf("unable to refresh the CDI registry: %v", err)
	}

	handler := &CDIHandler{
		registry: registry,
	}

	return handler, nil
}

func (cdi *CDIHandler) CreateCDISpecFile(claimUID string) error {
	specName := cdiapi.GenerateTransientSpecName(cdiVendor, cdiClass, claimUID)

	spec := &cdispec.Spec{
		Kind: cdiKind,
		Devices: []cdispec.Device{
			{
				Name: claimUID,
				ContainerEdits: cdispec.ContainerEdits{
					Env: []string{
						fmt.Sprintf("NETWORK_DEVICE=%s", "test-abc"),
					},
					Hooks: []*cdispec.Hook{
						{
							HookName: "createContainer",
							// Path:     "/test.sh",
							// Args:     []string{},
						},
					},
				},
			},
		},
	}

	minVersion, err := cdiapi.MinimumRequiredVersion(spec)
	if err != nil {
		return fmt.Errorf("failed to get minimum required CDI spec version: %v", err)
	}
	spec.Version = minVersion

	return cdi.registry.SpecDB().WriteSpec(spec, specName)
}

func (cdi *CDIHandler) GetClaimDevices(claimUID string) ([]string, error) {
	cdiDevices := []string{
		parser.QualifiedName(cdiVendor, cdiClass, claimUID),
	}

	return cdiDevices, nil
}
