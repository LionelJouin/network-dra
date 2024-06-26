/*
Copyright (c) 2024

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "github.com/LionelJouin/network-dra/api/dra.networking/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// NetworkAttachmentLister helps list NetworkAttachments.
// All objects returned here must be treated as read-only.
type NetworkAttachmentLister interface {
	// List lists all NetworkAttachments in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.NetworkAttachment, err error)
	// NetworkAttachments returns an object that can list and get NetworkAttachments.
	NetworkAttachments(namespace string) NetworkAttachmentNamespaceLister
	NetworkAttachmentListerExpansion
}

// networkAttachmentLister implements the NetworkAttachmentLister interface.
type networkAttachmentLister struct {
	indexer cache.Indexer
}

// NewNetworkAttachmentLister returns a new NetworkAttachmentLister.
func NewNetworkAttachmentLister(indexer cache.Indexer) NetworkAttachmentLister {
	return &networkAttachmentLister{indexer: indexer}
}

// List lists all NetworkAttachments in the indexer.
func (s *networkAttachmentLister) List(selector labels.Selector) (ret []*v1alpha1.NetworkAttachment, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.NetworkAttachment))
	})
	return ret, err
}

// NetworkAttachments returns an object that can list and get NetworkAttachments.
func (s *networkAttachmentLister) NetworkAttachments(namespace string) NetworkAttachmentNamespaceLister {
	return networkAttachmentNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// NetworkAttachmentNamespaceLister helps list and get NetworkAttachments.
// All objects returned here must be treated as read-only.
type NetworkAttachmentNamespaceLister interface {
	// List lists all NetworkAttachments in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.NetworkAttachment, err error)
	// Get retrieves the NetworkAttachment from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1alpha1.NetworkAttachment, error)
	NetworkAttachmentNamespaceListerExpansion
}

// networkAttachmentNamespaceLister implements the NetworkAttachmentNamespaceLister
// interface.
type networkAttachmentNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all NetworkAttachments in the indexer for a given namespace.
func (s networkAttachmentNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.NetworkAttachment, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.NetworkAttachment))
	})
	return ret, err
}

// Get retrieves the NetworkAttachment from the indexer for a given namespace and name.
func (s networkAttachmentNamespaceLister) Get(name string) (*v1alpha1.NetworkAttachment, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("networkattachment"), name)
	}
	return obj.(*v1alpha1.NetworkAttachment), nil
}
