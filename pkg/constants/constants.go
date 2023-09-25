// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package constants

const (
	ClusterName           = "name"
	DisplayName           = "display-name"
	KubernetesVersion     = "kubernetes-version"
	NodePublicKeyContents = "node-public-key-contents"

	CompartmentID      = "compartment-id"
	QuickCreateVCN     = "quick-create-vcn"
	CNIType            = "cni-type"
	VcnID              = "vcn-id"
	WorkerNodeSubnet   = "worker-node-subnet"
	ControlPlaneSubnet = "control-plane-subnet"
	LoadBalancerSubnet = "load-balancer-subnet"
	PodSubnet          = "pod-subnet"
	PodCIDR            = "pod-cidr"
	ClusterCIDR        = "cluster-cidr"
	ImageDisplayName   = "image-display-name"
	ImageId            = "image-id"

	RawNodePools = "node-pools"
	ApplyYAMLs   = "apply-yamls"

	CloudCredentialId = "cloud-credential-id"
	Region            = "region"

	InstallVerrazzano  = "install-verrazzano"
	VerrazzanoResource = "verrazzano-resource"
	VerrazzanoVersion  = "verrazzano-version"
)
