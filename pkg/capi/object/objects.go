// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package object

import (
	"fmt"
	"github.com/verrazzano/kontainer-engine-driver-oke-capi/pkg/templates"
	"github.com/verrazzano/kontainer-engine-driver-oke-capi/pkg/variables"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"strings"
)

//GVR attempts to find the GVR for an unstructured object

func DefaultingNamespace(u *unstructured.Unstructured) string {
	ns := u.GetNamespace()
	if len(ns) > 0 {
		return ns
	}
	return "default"
}

func GVR(u *unstructured.Unstructured) schema.GroupVersionResource {
	gvk := u.GroupVersionKind()

	kind := strings.ToLower(gvk.Kind)
	var resource string
	if kind[len(kind)-1] == 'y' {
		resource = strings.TrimSuffix(kind, "y") + "ies"
	} else {
		resource = kind + "s"
	}
	return schema.GroupVersionResource{
		Group:   gvk.Group,
		Version: gvk.Version,
		// e.g., "Verrazzano" becomes "verrazzanos"
		Resource: resource,
	}
}

func NestedField(o interface{}, fields ...string) (interface{}, error) {
	if len(fields) < 1 {
		return o, nil
	}
	field, remainingFields := fields[0], fields[1:]

	oMap, isMap := o.(map[string]interface{})
	if !isMap {
		return nil, fmt.Errorf("%v is not a map", o)
	}
	oNew, ok := oMap[field]
	if !ok {
		return nil, fmt.Errorf("field %s not found", field)
	}
	return NestedField(oNew, remainingFields...)
}

func Modules(v *variables.Variables) []Object {
	var objects []Object

	return objects
}

func CreateObjects() []Object {
	return objectList(include{
		workers:      true,
		controlplane: true,
		capi:         true,
	})
}

func UpdateObjects() []Object {
	return objectList(include{
		workers:      false,
		controlplane: false,
		capi:         true,
	})
}

func objectList(i include) []Object {
	var res []Object

	if i.capi {
		res = append(res, capi...)
	}
	if i.controlplane {
		res = append(res, ControlPlane...)
	}
	if i.workers {
		res = append(res, Workers...)
	}
	return res
}

type Object struct {
	Text         string
	LockedFields map[string]bool
}

type include struct {
	workers      bool
	controlplane bool
	capi         bool
}

var ControlPlane = []Object{
	{Text: templates.OCIManagedControlPlane},
}

var Workers = []Object{
	{Text: templates.MachinePool},
	{Text: templates.OCIManagedMachinePool},
}

var capi = []Object{
	CAPICluster,
	{Text: templates.ClusterIdentity},
	{
		Text: templates.OCIManagedCluster,
		LockedFields: map[string]bool{
			"networkSpec": true,
		},
	},
}

var CAPICluster = Object{Text: templates.Cluster}
