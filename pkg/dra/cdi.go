package dra

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/LionelJouin/network-dra/api/dra.networking/v1alpha1"
	cdiapi "github.com/container-orchestrated-devices/container-device-interface/pkg/cdi"
	cdispec "github.com/container-orchestrated-devices/container-device-interface/specs-go"
	drapbv1 "k8s.io/kubelet/pkg/apis/dra/v1alpha3"
	"tags.cncf.io/container-device-interface/pkg/parser"
)

const (
	cdiVendor = "k8s." + v1alpha1.GroupName
	cdiClass  = "attachment"
	cdiKind   = cdiVendor + "/" + cdiClass
)

type CDIHandler struct {
	OCIHookPath       string
	OCIHookSocketPath string
	Registry          cdiapi.Registry
}

func NewCDIHandler(cdiRoot string, ociHookPath string, ociHookSocketPath string) (*CDIHandler, error) {
	info, err := os.Stat(cdiRoot)
	switch {
	case err != nil && os.IsNotExist(err):
		err := os.MkdirAll(cdiRoot, 0o750)
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
		OCIHookPath:       ociHookPath,
		OCIHookSocketPath: ociHookSocketPath,
		Registry:          registry,
	}

	return handler, nil
}

func (cdi *CDIHandler) CreateCDISpecFile(claim *drapbv1.Claim, claimSpec *ClaimParams) error {
	specName := cdiapi.GenerateTransientSpecName(cdiVendor, cdiClass, claim.Uid)

	claimSpecStr, err := claimSpec.encode()
	if err != nil {
		return err
	}

	spec := &cdispec.Spec{
		Kind: cdiKind,
		Devices: []cdispec.Device{
			{
				Name: claim.Uid,
				ContainerEdits: cdispec.ContainerEdits{
					Env: []string{
						fmt.Sprintf("NETWORK_DEVICE=%s", "test-abc"),
					},
					Hooks: []*cdispec.Hook{
						{
							HookName: "createRuntime",
							Path:     cdi.OCIHookPath,
							Args: []string{
								filepath.Base(cdi.OCIHookPath),
								"run",
								fmt.Sprintf("--claim-uid=%s", claim.Uid),
								fmt.Sprintf("--claim-name=%s", claim.Name),
								fmt.Sprintf("--claim-namespace=%s", claim.Namespace),
								fmt.Sprintf("--oci-hook-socket-path=%s", cdi.OCIHookSocketPath),
								fmt.Sprintf("--claim-spec=%s", claimSpecStr),
							},
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

	return cdi.Registry.SpecDB().WriteSpec(spec, specName)
}

func (cdi *CDIHandler) GetClaimDevices(claimUID string) ([]string, error) {
	cdiDevices := []string{
		parser.QualifiedName(cdiVendor, cdiClass, claimUID),
	}

	return cdiDevices, nil
}
