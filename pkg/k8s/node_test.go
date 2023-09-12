// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8s

import (
	"context"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"testing"
)

func TestSetSingleNodeTaints(t *testing.T) {
	ctx := context.TODO()
	workerNode := createTestNode("worker")
	masterNode := createTestNode("master", v1.Taint{
		Key:    controlPlaneTaint,
		Effect: "NoSchedule",
	}, v1.Taint{
		Key:    masterTaint,
		Effect: "NoSchedule",
	}, v1.Taint{
		Key:    "abc",
		Effect: "xyz",
	})
	ki := fake.NewSimpleClientset(workerNode, masterNode)

	err := SetSingleNodeTaints(ctx, ki)
	assert.NoError(t, err)

	nodes, err := ki.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Len(t, nodes.Items, 2)
	for _, node := range nodes.Items {
		for _, taint := range node.Spec.Taints {
			assert.NotEqual(t, controlPlaneTaint, taint.Key)
			assert.NotEqual(t, masterTaint, taint.Key)
		}
	}
}

func createTestNode(name string, taints ...v1.Taint) *v1.Node {
	return &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1.NodeSpec{
			Taints: taints,
		},
	}
}
