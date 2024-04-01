package dra

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/LionelJouin/network-dra/api/dra.networking/v1alpha1"
	netdefv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
)

type ClaimParams struct {
	NetworkAttachmentSpec           *v1alpha1.NetworkAttachmentSpec           `json:"network-attachment-spec"`
	NetworkAttachmentDefinitionSpec *netdefv1.NetworkAttachmentDefinitionSpec `json:"network-attachment-definition-spec"`
}

// https://stackoverflow.com/questions/63126139/how-to-convert-struct-to-base64-encoded-string-and-viceversa
func (cp *ClaimParams) encode() (string, error) {
	var buf bytes.Buffer

	encoder := base64.NewEncoder(base64.StdEncoding, &buf)

	err := json.NewEncoder(encoder).Encode(cp)
	if err != nil {
		return "", err
	}

	encoder.Close()

	return buf.String(), nil
}

func (cp *ClaimParams) decode(enc string) error {
	return json.NewDecoder(base64.NewDecoder(base64.StdEncoding, strings.NewReader(enc))).Decode(cp)
}
