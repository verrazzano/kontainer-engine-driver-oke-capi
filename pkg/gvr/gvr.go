// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package gvr

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	InfrastructureXK8sIO = "infrastructure.cluster.x-k8s.io"
	ClusterXK8sIO        = "cluster.x-k8s.io"
	V1Beta1Version       = "v1beta1"
	V1Beta2Version       = "v1beta2"
)

var Cluster = schema.GroupVersionResource{
	Group:    ClusterXK8sIO,
	Version:  V1Beta1Version,
	Resource: "clusters",
}

var OCICluster = schema.GroupVersionResource{
	Group:    InfrastructureXK8sIO,
	Version:  V1Beta2Version,
	Resource: "ocimanagedclusters",
}

var ClusterIdentity = schema.GroupVersionResource{
	Group:    InfrastructureXK8sIO,
	Version:  V1Beta1Version,
	Resource: "ociclusteridentities",
}

var MachinePool = schema.GroupVersionResource{
	Group:    ClusterXK8sIO,
	Version:  V1Beta1Version,
	Resource: "machinepools",
}

var OCIMachinePools = schema.GroupVersionResource{
	Group:    InfrastructureXK8sIO,
	Version:  V1Beta2Version,
	Resource: "ocimanagedmachinepools",
}
