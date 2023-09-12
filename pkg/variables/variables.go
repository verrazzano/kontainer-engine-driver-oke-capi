// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package variables

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/rancher/kontainer-engine/drivers/options"
	"github.com/rancher/kontainer-engine/store"
	"github.com/rancher/kontainer-engine/types"
	driverconst "github.com/verrazzano/kontainer-engine-driver-oke-capi/pkg/constants"
	"github.com/verrazzano/kontainer-engine-driver-oke-capi/pkg/gvr"
	"github.com/verrazzano/kontainer-engine-driver-oke-capi/pkg/k8s"
	"github.com/verrazzano/kontainer-engine-driver-oke-capi/pkg/oci"
	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"strings"
)

const (
	DefaultOCICPUs                 = 2
	DefaultMemoryGbs               = 16
	DefaultVolumeGbs               = 100
	DefaultNodePVTransitEncryption = true
	DefaultVMShape                 = "VM.Standard.E4.Flex"
	ProviderId                     = `oci://{{ ds["id"] }}`
)

const (
	kubeconfigName = "%s-kubeconfig"

	loadBalancerSubnetRole         = "service-lb"
	controlPlaneEndpointSubnetRole = "control-plane-endpoint"
	workerSubnetRole               = "worker"
)

type Subnet struct {
	Id   string
	Role string
	Name string
	CIDR string
	Type string
}

type NodePool struct {
	Name       string `json:"name"`
	Replicas   int64  `json:"replicas"`
	Memory     int64  `json:"memory"`
	Ocpus      int64  `json:"ocpus"`
	VolumeSize int64  `json:"volumeSize"`
	Shape      string `json:"shape"`
	Version    string `json:"version"`
}

var OCIClientGetter = func(v *Variables) (oci.Client, error) {
	return oci.NewClient(v.GetConfigurationProvider())
}

type (
	//Variables are parameters for cluster lifecycle operations
	Variables struct {
		Name             string
		DisplayName      string
		Namespace        string
		Hash             string
		ControlPlaneHash string
		NodePoolHash     string

		QuickCreateVCN     bool
		VCNID              string
		WorkerNodeSubnet   string
		ControlPlaneSubnet string
		LoadBalancerSubnet string
		// Parsed subnets
		Subnets     []Subnet `json:"subnets,omitempty"`
		PodCIDR     string
		ClusterCIDR string

		// Cluster topology and configuration
		KubernetesVersion string
		SSHPublicKey      string
		RawNodePools      []string
		ApplyYAMLS        []string
		// Parsed node pools
		NodePools []NodePool

		// ImageID is looked up by display name
		ImageDisplayName string
		ImageID          string
		ActualImage      string

		// OCI Credentials
		CloudCredentialId    string
		CompartmentID        string
		Fingerprint          string
		PrivateKey           string
		PrivateKeyPassphrase string
		Region               string
		Tenancy              string
		User                 string

		// Supplied for templating
		ProviderId string
	}
)

// NewFromOptions creates a new Variables given *types.DriverOptions
func NewFromOptions(ctx context.Context, driverOptions *types.DriverOptions) (*Variables, error) {
	v := &Variables{
		Name:              options.GetValueFromDriverOptions(driverOptions, types.StringType, driverconst.ClusterName).(string),
		DisplayName:       options.GetValueFromDriverOptions(driverOptions, types.StringType, driverconst.DisplayName, "displayName").(string),
		KubernetesVersion: options.GetValueFromDriverOptions(driverOptions, types.StringType, driverconst.KubernetesVersion, "kubernetesVersion").(string),

		// User and authentication
		SSHPublicKey:      options.GetValueFromDriverOptions(driverOptions, types.StringType, driverconst.NodePublicKeyContents, "nodePublicKeyContents").(string),
		CloudCredentialId: options.GetValueFromDriverOptions(driverOptions, types.StringType, driverconst.CloudCredentialId, "cloudCredentialId").(string),
		Region:            options.GetValueFromDriverOptions(driverOptions, types.StringType, driverconst.Region, "region").(string),
		CompartmentID:     options.GetValueFromDriverOptions(driverOptions, types.StringType, driverconst.CompartmentID, "compartmentId").(string),

		// Networking
		QuickCreateVCN:     options.GetValueFromDriverOptions(driverOptions, types.BoolType, driverconst.QuickCreateVCN, "quickCreateVcn").(bool),
		VCNID:              options.GetValueFromDriverOptions(driverOptions, types.StringType, driverconst.VcnID, "vcnId").(string),
		WorkerNodeSubnet:   options.GetValueFromDriverOptions(driverOptions, types.StringType, driverconst.WorkerNodeSubnet, "workerNodeSubnet").(string),
		LoadBalancerSubnet: options.GetValueFromDriverOptions(driverOptions, types.StringType, driverconst.LoadBalancerSubnet, "loadBalancerSubnet").(string),
		ControlPlaneSubnet: options.GetValueFromDriverOptions(driverOptions, types.StringType, driverconst.ControlPlaneSubnet, "controlPlaneSubnet").(string),
		PodCIDR:            options.GetValueFromDriverOptions(driverOptions, types.StringType, driverconst.PodCIDR, "podCidr").(string),
		ClusterCIDR:        options.GetValueFromDriverOptions(driverOptions, types.StringType, driverconst.ClusterCIDR, "clusterCidr").(string),

		// VM settings
		ImageDisplayName: options.GetValueFromDriverOptions(driverOptions, types.StringType, driverconst.ImageDisplayName, "imageDisplayName").(string),
		RawNodePools:     options.GetValueFromDriverOptions(driverOptions, types.StringSliceType, driverconst.RawNodePools, "nodePools").(*types.StringSlice).Value,
		ApplyYAMLS:       options.GetValueFromDriverOptions(driverOptions, types.StringSliceType, driverconst.ApplyYAMLs, "applyYamls").(*types.StringSlice).Value,

		ImageID:    options.GetValueFromDriverOptions(driverOptions, types.StringType, driverconst.ImageId, "imageId").(string),
		ProviderId: ProviderId,
	}
	v.Namespace = v.Name

	if err := v.SetDynamicValues(ctx); err != nil {
		return v, err
	}
	return v, nil
}

// SetUpdateValues are the values potentially changed during an update operation
func (v *Variables) SetUpdateValues(ctx context.Context, vNew *Variables) error {
	v.KubernetesVersion = vNew.KubernetesVersion
	v.ImageDisplayName = vNew.ImageDisplayName
	v.RawNodePools = vNew.RawNodePools
	v.SSHPublicKey = vNew.SSHPublicKey
	v.DisplayName = vNew.DisplayName
	v.ImageID = vNew.ImageID
	v.ApplyYAMLS = vNew.ApplyYAMLS
	return v.SetDynamicValues(ctx)
}

// SetDynamicValues sets dynamic values
func (v *Variables) SetDynamicValues(ctx context.Context) error {
	// deserialize node pools
	nodePools, err := v.ParseNodePools()
	if err != nil {
		return err
	}
	v.NodePools = nodePools

	// setup OCI client for dynamic values
	ki, err := k8s.InjectedInterface()
	if err != nil {
		return err
	}
	if err := SetupOCIAuth(ctx, ki, v); err != nil {
		return err
	}
	ociClient, err := OCIClientGetter(v)

	if err != nil {
		return err
	}
	// get image OCID from OCI
	if err := v.setImageId(ctx, ociClient); err != nil {
		return err
	}
	// get subnet metadata from OCI
	if err := v.setSubnets(ctx, ociClient); err != nil {
		return err
	}

	// set hashes for controlplane updates
	v.SetHashes()
	return nil
}

// GetConfigurationProvider creates a new configuration provider from Variables
func (v *Variables) GetConfigurationProvider() common.ConfigurationProvider {
	var passphrase *string
	if len(v.PrivateKeyPassphrase) > 0 {
		passphrase = &v.PrivateKeyPassphrase
	}
	privateKey := strings.TrimSpace(v.PrivateKey)
	return common.NewRawConfigurationProvider(v.Tenancy, v.User, v.Region, v.Fingerprint, privateKey, passphrase)
}

// GetCAPIClusterKubeConfig fetches the cluster's kubeconfig
func (v *Variables) GetCAPIClusterKubeConfig(ctx context.Context) (*store.KubeConfig, error) {
	client, err := k8s.InjectedInterface()
	if err != nil {
		return nil, err
	}
	kubeconfigSecretName := fmt.Sprintf(kubeconfigName, v.Name)
	secret, err := client.CoreV1().Secrets(v.Namespace).Get(ctx, kubeconfigSecretName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	kubeconfig := &store.KubeConfig{}
	err = yaml.Unmarshal(secret.Data["value"], kubeconfig)
	if err != nil {
		return nil, err
	}
	return kubeconfig, nil
}

// NodeCount is the sum of worker nodes
func (v *Variables) NodeCount() (*types.NodeCount, error) {
	nps, err := v.ParseNodePools()
	if err != nil {
		return nil, err
	}
	v.NodePools = nps
	return &types.NodeCount{
		Count: v.workerNodeCount(),
	}, nil
}

func (v *Variables) IsSingleNodeCluster() bool {
	return v.workerNodeCount() == 0
}

func (v *Variables) workerNodeCount() int64 {
	var count int64 = 0
	for _, np := range v.NodePools {
		count = count + np.Replicas
	}
	return count
}

// Version is the cluster Kubernetes version
func (v *Variables) Version() *types.KubernetesVersion {
	return &types.KubernetesVersion{
		Version: v.KubernetesVersion,
	}
}

func (v *Variables) ParseNodePools() ([]NodePool, error) {
	var nodePools []NodePool

	for _, rawNodePool := range v.RawNodePools {
		nodePool := NodePool{}
		if err := json.Unmarshal([]byte(rawNodePool), &nodePool); err != nil {
			return nil, err
		}
		nodePools = append(nodePools, nodePool)
	}

	return nodePools, nil
}

func (v *Variables) setImageId(ctx context.Context, client oci.Client) error {
	// if user is bringing their own image, skip the dynamic image lookup

	imageId, err := client.GetImageIdByName(ctx, v.ImageDisplayName, v.CompartmentID)
	if err != nil {
		return err
	}
	v.ActualImage = imageId

	return nil
}

func (v *Variables) setSubnets(ctx context.Context, client oci.Client) error {
	var subnets []Subnet
	subnetCache := map[string]*Subnet{}

	addSubnetForRole := func(subnetId, role string) error {
		var err error
		subnet := subnetCache[subnetId]
		if subnet == nil && subnetId != "" {
			subnet, err = getSubnetById(ctx, client, subnetId, role)
			if err != nil {
				return err
			}
		}
		if subnet != nil {
			subnets = append(subnets, *subnet)
		}
		return nil
	}

	if err := addSubnetForRole(v.LoadBalancerSubnet, loadBalancerSubnetRole); err != nil {
		return err
	}
	if err := addSubnetForRole(v.WorkerNodeSubnet, workerSubnetRole); err != nil {
		return err
	}
	if err := addSubnetForRole(v.ControlPlaneSubnet, controlPlaneEndpointSubnetRole); err != nil {
		return err
	}
	v.Subnets = subnets
	return nil
}

func getSubnetById(ctx context.Context, client oci.Client, subnetId, role string) (*Subnet, error) {
	sn, err := client.GetSubnetById(ctx, subnetId)
	if err != nil {
		return nil, fmt.Errorf("failed to get subnet %s", subnetId)
	}

	return &Subnet{
		Id:   subnetId,
		CIDR: *sn.CidrBlock,
		Type: oci.SubnetAccess(*sn),
		Name: role,
		Role: role,
	}, nil
}

// SetupOCIAuth dynamically loads OCI authentication
func SetupOCIAuth(ctx context.Context, client kubernetes.Interface, v *Variables) error {
	ccName, ccNamespace := v.cloudCredentialNameAndNamespace()
	cc, err := client.CoreV1().Secrets(ccNamespace).Get(ctx, ccName, metav1.GetOptions{})
	// Failed to retrieve cloud credentials
	if err != nil {
		return err
	}

	v.User = string(cc.Data["ocicredentialConfig-userId"])
	v.Fingerprint = string(cc.Data["ocicredentialConfig-fingerprint"])
	v.Tenancy = string(cc.Data["ocicredentialConfig-tenancyId"])
	v.PrivateKeyPassphrase = string(cc.Data["ocicredentialConfig-passphrase"])
	v.PrivateKey = string(cc.Data["ocicredentialConfig-privateKeyContents"])
	return nil
}

func (v *Variables) SetQuickCreateVCNInfo(ctx context.Context, di dynamic.Interface) error {
	// Only set Quick Create VCN Info if using Quick Create VCN, and the VCN info is unset.
	if v.QuickCreateVCN && v.isNetworkingUnset() {
		ociCluster, err := di.Resource(gvr.OCICluster).Namespace(v.Name).Get(ctx, v.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		// Get VCN id and VCN subnets from the OCI Cluster resource
		vcnID, found, err := unstructured.NestedString(ociCluster.Object, "spec", "networkSpec", "vcn", "id")
		if err != nil || !found {
			return errors.New("waiting for VCN to be created")
		}
		v.VCNID = vcnID
		subnets, found, err := unstructured.NestedSlice(ociCluster.Object, "spec", "networkSpec", "vcn", "subnets")
		if err != nil || !found {
			return errors.New("waiting for subnets to be created")
		}

		// For each subnet in the VCN subnet list, identify its role and populate the subnet id in the cluster state
		for _, subnet := range subnets {
			subnetObject, ok := subnet.(map[string]interface{})
			if !ok {
				return errors.New("subnet is creating")
			}

			// Get nested subnet Role and id from the subnet object
			subnetRole, found, err := unstructured.NestedString(subnetObject, "role")
			if err != nil || !found {
				return errors.New("waiting for subnet role to be populated")
			}
			subnetId, found, err := unstructured.NestedString(subnetObject, "id")
			if err != nil || !found {
				return errors.New("waiting for subnet id to be populated")
			}

			// Populate the subnet id depending on the subnet role
			switch subnetRole {
			case controlPlaneEndpointSubnetRole:
				v.ControlPlaneSubnet = subnetId
			case loadBalancerSubnetRole:
				v.LoadBalancerSubnet = subnetId
			case workerSubnetRole:
				v.WorkerNodeSubnet = subnetId
			default: // we are not interested in any other subnets
				continue
			}
		}
	}

	return nil
}

func (v *Variables) isNetworkingUnset() bool {
	return len(v.VCNID) < 1 || len(v.ControlPlaneSubnet) < 1 || len(v.LoadBalancerSubnet) < 1 || len(v.WorkerNodeSubnet) < 1
}

func (v *Variables) cloudCredentialNameAndNamespace() (string, string) {
	split := strings.Split(v.CloudCredentialId, ":")

	if len(split) == 1 {
		return "cattle-global-data", split[0]
	}
	return split[1], split[0]
}
