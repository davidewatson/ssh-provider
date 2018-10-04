/*
Copyright 2018 Platform 9 Systems, Inc.

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

// This file was automatically generated by lister-gen

package v1alpha2

import (
	v1alpha2 "github.com/platform9/ssh-provider/pkg/apis/sshprovider/v1alpha2"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// ProvisionedMachineLister helps list ProvisionedMachines.
type ProvisionedMachineLister interface {
	// List lists all ProvisionedMachines in the indexer.
	List(selector labels.Selector) (ret []*v1alpha2.ProvisionedMachine, err error)
	// ProvisionedMachines returns an object that can list and get ProvisionedMachines.
	ProvisionedMachines(namespace string) ProvisionedMachineNamespaceLister
	ProvisionedMachineListerExpansion
}

// provisionedMachineLister implements the ProvisionedMachineLister interface.
type provisionedMachineLister struct {
	indexer cache.Indexer
}

// NewProvisionedMachineLister returns a new ProvisionedMachineLister.
func NewProvisionedMachineLister(indexer cache.Indexer) ProvisionedMachineLister {
	return &provisionedMachineLister{indexer: indexer}
}

// List lists all ProvisionedMachines in the indexer.
func (s *provisionedMachineLister) List(selector labels.Selector) (ret []*v1alpha2.ProvisionedMachine, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha2.ProvisionedMachine))
	})
	return ret, err
}

// ProvisionedMachines returns an object that can list and get ProvisionedMachines.
func (s *provisionedMachineLister) ProvisionedMachines(namespace string) ProvisionedMachineNamespaceLister {
	return provisionedMachineNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// ProvisionedMachineNamespaceLister helps list and get ProvisionedMachines.
type ProvisionedMachineNamespaceLister interface {
	// List lists all ProvisionedMachines in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*v1alpha2.ProvisionedMachine, err error)
	// Get retrieves the ProvisionedMachine from the indexer for a given namespace and name.
	Get(name string) (*v1alpha2.ProvisionedMachine, error)
	ProvisionedMachineNamespaceListerExpansion
}

// provisionedMachineNamespaceLister implements the ProvisionedMachineNamespaceLister
// interface.
type provisionedMachineNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all ProvisionedMachines in the indexer for a given namespace.
func (s provisionedMachineNamespaceLister) List(selector labels.Selector) (ret []*v1alpha2.ProvisionedMachine, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha2.ProvisionedMachine))
	})
	return ret, err
}

// Get retrieves the ProvisionedMachine from the indexer for a given namespace and name.
func (s provisionedMachineNamespaceLister) Get(name string) (*v1alpha2.ProvisionedMachine, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha2.Resource("provisionedmachine"), name)
	}
	return obj.(*v1alpha2.ProvisionedMachine), nil
}