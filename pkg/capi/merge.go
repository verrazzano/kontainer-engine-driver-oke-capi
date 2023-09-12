// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capi

import "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

func mergeUnstructured(base *unstructured.Unstructured, merge *unstructured.Unstructured, lockedFields map[string]bool) *unstructured.Unstructured {
	merged := mergeMaps(base.Object, merge.Object, lockedFields)
	return &unstructured.Unstructured{
		Object: merged,
	}
}

func mergeMaps(m1, m2 map[string]interface{}, lockedFields map[string]bool) map[string]interface{} {
	for k, v2 := range m2 {
		// don't update locked fields
		if lockedFields[k] {
			continue
		}
		if vm1, vm2, ok := isRecursiveMerge(k, m1, v2); ok {
			// recursively merge maps if both values are maps
			m1[k] = mergeMaps(vm1, vm2, lockedFields)
		} else {
			// otherwise replace key
			m1[k] = v2
		}
	}
	return m1
}

func isRecursiveMerge(k string, m1 map[string]interface{}, v2 interface{}) (map[string]interface{}, map[string]interface{}, bool) {
	v1, ok := m1[k]
	if !ok {
		return nil, nil, false
	}
	v1Map, isv1Map := v1.(map[string]interface{})
	if !isv1Map {
		return nil, nil, false
	}
	v2Map, isv2MAp := v2.(map[string]interface{})
	if !isv2MAp {
		return nil, nil, false
	}
	return v1Map, v2Map, true
}
