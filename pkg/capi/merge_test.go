// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capi

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMerge(t *testing.T) {
	var tests = []struct {
		name         string
		m1           map[string]interface{}
		m2           map[string]interface{}
		res          map[string]interface{}
		lockedFields map[string]bool
	}{
		{
			"merge two basic maps",
			map[string]interface{}{
				"a": "b",
				"nest": map[string]interface{}{
					"v": "w",
					"x": "y",
				},
			},
			map[string]interface{}{
				"1": 2,
				"nest": map[string]interface{}{
					"x": "z",
				},
			},
			map[string]interface{}{
				"1": 2,
				"a": "b",
				"nest": map[string]interface{}{
					"v": "w",
					"x": "z",
				},
			},
			nil,
		},
		{
			"merge two basic maps with locked fields",
			map[string]interface{}{
				"a": "b",
				"nest": map[string]interface{}{
					"v": "w",
					"x": "y",
				},
			},
			map[string]interface{}{
				"1": 2,
				"nest": map[string]interface{}{
					"x": "z",
				},
			},
			map[string]interface{}{
				"1": 2,
				"a": "b",
				"nest": map[string]interface{}{
					"v": "w",
					"x": "y",
				},
			},
			map[string]bool{
				"x": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := mergeMaps(tt.m1, tt.m2, tt.lockedFields)
			assert.EqualValues(t, tt.res, res)
		})
	}
}
