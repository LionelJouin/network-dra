package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NetworkAttachement is a specification for a NetworkAttachement resource.
type NetworkAttachement struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Specification of the desired behavior of the NetworkAttachement.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
	// +optional
	Spec NetworkAttachementSpec `json:"spec"`

	// Most recently observed status of the NetworkAttachement.
	// Populated by the system.
	// Read-only.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
	// +optional
	Status NetworkAttachementStatus `json:"status"`
}

// NetworkAttachementSpec is the spec for a NetworkAttachement resource.
type NetworkAttachementSpec struct{}

// NetworkAttachementStatus is the status for a NetworkAttachement resource.
type NetworkAttachementStatus struct{}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NetworkAttachementList is a list of NetworkAttachement resources.
type NetworkAttachementList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []NetworkAttachement `json:"items"`
}
