// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8s

import (
	"context"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	controlPlaneTaint = "node-role.kubernetes.io/control-plane"
	masterTaint       = "node-role.kubernetes.io/master"
)

// SetSingleNodeTaints will mark control plane nodes for scheduling by removing taints.
func SetSingleNodeTaints(ctx context.Context, ki kubernetes.Interface) error {
	nodes, err := ki.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, node := range nodes.Items {
		var taints []v1.Taint
		for _, taint := range node.Spec.Taints {
			if !isControlPlaneNoScheduleTaint(taint) {
				taints = append(taints, taint)
			}
		}
		node.Spec.Taints = taints
		delete(node.Labels, "node.kubernetes.io/exclude-from-external-load-balancers")
		_, err = ki.CoreV1().Nodes().Update(ctx, &node, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}

func isControlPlaneNoScheduleTaint(taint v1.Taint) bool {
	return taint.Key == controlPlaneTaint || taint.Key == masterTaint
}
