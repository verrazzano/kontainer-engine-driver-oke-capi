// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capi

import (
	"context"
	"errors"
	"github.com/verrazzano/kontainer-engine-driver-oke-capi/pkg/capi/object"
	"github.com/verrazzano/kontainer-engine-driver-oke-capi/pkg/gvr"
	"github.com/verrazzano/kontainer-engine-driver-oke-capi/pkg/provisioning"
	"github.com/verrazzano/kontainer-engine-driver-oke-capi/pkg/variables"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
)

func IsCAPIClusterReady(ctx context.Context, client dynamic.Interface, state *variables.Variables, plog *provisioning.Logger) error {
	cluster, err := client.Resource(gvr.Cluster).Namespace(state.Namespace).Get(ctx, state.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if isClusterReady(cluster, state) {
		poolsReady := true
		if len(state.NodePools) > 0 {
			poolsReady, err = areMachinePoolsReady(ctx, client, state)
			if err != nil {
				_ = plog.ClusterStatus(cluster)
				return errors.New("Waiting for nodes to be ready")
			}
		}
		if poolsReady {
			return nil
		}
	}
	_ = plog.ClusterStatus(cluster)
	return errors.New("Waiting for cluster to be ready")
}

func isClusterReady(cluster *unstructured.Unstructured, state *variables.Variables) bool {
	switch state.KubernetesVersion {
	case "v1.25.4":
		controlPlaneReady, err := object.NestedField(cluster.Object, "status", "controlPlaneReady")
		if err != nil {
			return false
		}
		controlPlaneReadyBool, ok := controlPlaneReady.(bool)
		if !ok || !controlPlaneReadyBool {
			return false
		}
	}
	infrastructureReady, err := object.NestedField(cluster.Object, "status", "infrastructureReady")
	if err != nil {
		return false
	}
	infrastructureReadyBool, ok := infrastructureReady.(bool)
	if !ok || !infrastructureReadyBool {
		return false
	}

	phase, err := object.NestedField(cluster.Object, "status", "phase")
	if err != nil {
		return false
	}
	phaseString, ok := phase.(string)
	if !ok || phaseString != clusterPhaseProvisioned {
		return false
	}
	return true
}

func areMachinePoolsReady(ctx context.Context, client dynamic.Interface, state *variables.Variables) (bool, error) {
	poolList, err := client.Resource(gvr.MachinePool).List(ctx, metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(&metav1.LabelSelector{
			MatchLabels: map[string]string{
				"cluster.x-k8s.io/cluster-name": state.Name,
			},
		}),
	})
	if err != nil {
		return false, err
	}

	if poolList == nil || len(poolList.Items) < 1 {
		return false, nil
	}

	for _, pool := range poolList.Items {
		phase, err := object.NestedField(pool.Object, "status", "phase")
		if err != nil {
			return false, nil
		}
		phaseString, ok := phase.(string)
		if !ok {
			return false, nil
		}
		if phaseString != machinePoolPhaseRunning {
			return false, nil
		}
	}
	return true, nil
}
