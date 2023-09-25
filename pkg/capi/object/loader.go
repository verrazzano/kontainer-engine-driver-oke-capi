// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package object

import (
	"bytes"
	"errors"
	"github.com/verrazzano/kontainer-engine-driver-oke-capi/pkg/variables"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	apiyaml "k8s.io/apimachinery/pkg/util/yaml"
	"strings"
	"text/template"
)

// ToObjects adapts a slice of yaml documents into an object array
func ToObjects(yamlDocuments []string) []Object {
	var objects []Object
	for _, document := range yamlDocuments {
		yamls := strings.Split(document, "---")
		for _, y := range yamls {
			objects = append(objects, Object{
				Text: y,
			})
		}
	}

	return objects
}

func LoadTextTemplate(o Object, variables variables.Variables) ([]unstructured.Unstructured, error) {
	templatedBytes, err := createTextTemplate(o, variables)
	if err != nil {
		return nil, err
	}
	u, err := ToUnstructured(templatedBytes)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func createTextTemplate(o Object, variables variables.Variables) ([]byte, error) {
	t, err := template.New("objectText").Funcs(template.FuncMap{
		"contains": strings.Contains,
		"nindent": func(indent int, s string) string {
			spacing := strings.Repeat(" ", indent)
			split := strings.FieldsFunc(s, func(r rune) bool {
				switch r {
				case '\n', '\v', '\f', '\r':
					return true
				default:
					return false
				}
			})
			sb := strings.Builder{}
			for i := 0; i < len(split); i++ {
				segment := split[i]
				sb.WriteString(spacing)
				sb.WriteString(segment)
				if i < len(split)-1 {
					sb.WriteRune('\n')
				}
			}

			return sb.String()
		},
	}).Parse(o.Text)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, variables); err != nil {
		return nil, err
	}
	templatedBytes := buf.Bytes()
	return templatedBytes, nil
}

func ToUnstructured(o []byte) ([]unstructured.Unstructured, error) {
	j, err := apiyaml.ToJSON(o)
	if err != nil {
		return nil, err
	}
	obj, err := runtime.Decode(unstructured.UnstructuredJSONScheme, j)
	if err != nil {
		return nil, err
	}
	if u, ok := obj.(*unstructured.Unstructured); ok {
		return []unstructured.Unstructured{*u}, nil
	}
	if us, ok := obj.(*unstructured.UnstructuredList); ok {
		return us.Items, nil
	}

	return nil, errors.New("unknown object type during unstructured serialization")
}
