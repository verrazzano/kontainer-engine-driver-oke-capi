// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package object

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNestedField(t *testing.T) {
	var tests = []struct {
		name     string
		o        map[string]interface{}
		fields   []string
		res      interface{}
		hasError bool
	}{
		{
			"can get nested value",
			map[string]interface{}{
				"a": map[string]interface{}{
					"b": "c",
				},
			},
			[]string{"a", "b"},
			"c",
			false,
		},
		{
			"fails to get field that doesn't exist",
			map[string]interface{}{},
			[]string{"x"},
			nil,
			true,
		},
		{
			"can get nested spec value",
			map[string]interface{}{
				"spec": map[string]interface{}{
					"template": map[string]interface{}{
						"spec": map[string]interface{}{
							"infrastructureRef": map[string]interface{}{
								"name": "xyz",
							},
						},
					},
				},
			},
			[]string{"spec", "template", "spec", "infrastructureRef", "name"},
			"xyz",
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := NestedField(tt.o, tt.fields...)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.EqualValues(t, tt.res, res)
			}
		})
	}
}
