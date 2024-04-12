package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NetworkAttachment is a specification for a NetworkAttachment resource.
type NetworkAttachment struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Specification of the desired behavior of the NetworkAttachment.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
	// +optional
	Spec NetworkAttachmentSpec `json:"spec"`

	// Most recently observed status of the NetworkAttachment.
	// Populated by the system.
	// Read-only.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
	// +optional
	Status NetworkAttachmentStatus `json:"status"`
}

// NetworkAttachmentSpec is the spec for a NetworkAttachment resource.
// https://github.com/k8snetworkplumbingwg/network-attachment-definition-client/blob/v1.6.0/pkg/apis/k8s.cni.cncf.io/v1/types.go#L139
type NetworkAttachmentSpec struct {
	// Name contains the name of the Network object this element selects
	Name string `json:"name"`
	// Namespace contains the optional namespace that the network referenced
	// by Name exists in
	Namespace string `json:"namespace,omitempty"`
	// IPRequest contains an optional requested IP addresses for this network
	// attachment
	IPRequest []string `json:"ips,omitempty"`
	// MacRequest contains an optional requested MAC address for this
	// network attachment
	MacRequest string `json:"mac,omitempty"`
	// InterfaceRequest contains an optional requested name for the
	// network interface this attachment will create in the container
	InterfaceRequest string `json:"interface,omitempty"`
	Default          bool   `json:"default,omitempty"`
}

// NetworkAttachmentStatus is the status for a NetworkAttachment resource.
type NetworkAttachmentStatus struct{}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NetworkAttachmentList is a list of NetworkAttachment resources.
type NetworkAttachmentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []NetworkAttachment `json:"items"`
}
