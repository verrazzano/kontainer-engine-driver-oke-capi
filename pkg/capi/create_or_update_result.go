// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capi

import (
	"github.com/verrazzano/kontainer-engine-driver-oke-capi/pkg/capi/object"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type NameAndNamespace struct {
	Name      string
	Namespace string
}

type CreateOrUpdateResult struct {
	result map[string]map[NameAndNamespace]bool
}

func NewCreateOrUpdateResult() *CreateOrUpdateResult {
	return &CreateOrUpdateResult{
		result: map[string]map[NameAndNamespace]bool{},
	}
}

func (c *CreateOrUpdateResult) Add(resource string, u *unstructured.Unstructured) {
	if u == nil {
		return
	}
	if _, ok := c.result[resource]; !ok {
		c.result[resource] = map[NameAndNamespace]bool{}
	}
	c.result[resource][NameAndNamespace{
		Name:      u.GetName(),
		Namespace: object.DefaultingNamespace(u),
	}] = true
}

func (c *CreateOrUpdateResult) Contains(resource string, u *unstructured.Unstructured) bool {
	if u == nil {
		return false
	}
	if _, ok := c.result[resource]; !ok {
		return false
	}
	return c.result[resource][NameAndNamespace{
		Name:      u.GetName(),
		Namespace: object.DefaultingNamespace(u),
	}]
}

func (c *CreateOrUpdateResult) Merge(c2 *CreateOrUpdateResult) {
	for k, v := range c2.result {
		c.result[k] = v
	}
}
