// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/rancher/kontainer-engine/types"
	"github.com/verrazzano/kontainer-engine-driver-oke-capi/pkg/capi"
	driverconst "github.com/verrazzano/kontainer-engine-driver-oke-capi/pkg/constants"
	"github.com/verrazzano/kontainer-engine-driver-oke-capi/pkg/k8s"
	"github.com/verrazzano/kontainer-engine-driver-oke-capi/pkg/provisioning"
	"github.com/verrazzano/kontainer-engine-driver-oke-capi/pkg/variables"
	"github.com/verrazzano/kontainer-engine-driver-oke-capi/pkg/version"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"time"
)

const (
	metadataKey = "state"
)

type OKEDriver struct {
	Logger             *zap.SugaredLogger
	driverCapabilities types.Capabilities
}

func NewDriver() types.Driver {
	driver := &OKEDriver{
		driverCapabilities: types.Capabilities{
			Capabilities: make(map[int64]bool),
		},
	}

	driver.driverCapabilities.AddCapability(types.GetVersionCapability)
	driver.driverCapabilities.AddCapability(types.SetVersionCapability)
	driver.driverCapabilities.AddCapability(types.GetClusterSizeCapability)
	driver.driverCapabilities.AddCapability(types.SetClusterSizeCapability)

	return driver
}

func (d *OKEDriver) Remove(ctx context.Context, info *types.ClusterInfo) error {
	d.Logger.Infof("capi.driver.Remove(...) called")
	state, err := d.loadVariables(info)
	if err != nil {
		return err
	}
	adminDi, err := k8s.InjectedDynamic()
	if err != nil {
		return fmt.Errorf("failed to created admin cluster dynamic client: %v", err)
	}
	adminKi, err := k8s.InjectedInterface()
	if err != nil {
		return fmt.Errorf("failed to created admin cluster client: %v", err)
	}
	capiClient := d.NewCAPIClient(provisioning.NewLogger(ctx, adminKi, state.Name))
	if err := capiClient.DeleteVerrazzanoResources(ctx, adminDi, state); err != nil {
		return err
	}
	return capiClient.DeleteCluster(ctx, adminDi, adminKi, state)
}

// GetDriverCreateOptions implements driver interface
func (d *OKEDriver) GetDriverCreateOptions(ctx context.Context) (*types.DriverFlags, error) {
	d.Logger.Infof("capi.driver.GetDriverCreateOptions(...) called")

	defaults, err := loadDefaults(ctx)
	if err != nil {
		return nil, err
	}

	driverFlag := types.DriverFlags{
		Options: make(map[string]*types.Flag),
	}
	driverFlag.Options[driverconst.ClusterName] = &types.Flag{
		Type:  types.StringType,
		Usage: "The generated name of the OKE CAPI Cluster",
	}
	driverFlag.Options[driverconst.DisplayName] = &types.Flag{
		Type:  types.StringType,
		Usage: "The display name of the OKE CAPI Cluster",
	}
	driverFlag.Options[driverconst.PodCIDR] = &types.Flag{
		Type:  types.StringType,
		Usage: "The Kubernetes Pod CIDR block",
		Default: &types.Default{
			DefaultString: "10.244.0.0/16",
		},
	}
	driverFlag.Options[driverconst.ClusterCIDR] = &types.Flag{
		Type:  types.StringType,
		Usage: "The Kubernetes Cluster CIDR block",
		Default: &types.Default{
			DefaultString: "10.96.0.0/16",
		},
	}

	driverFlag.Options[driverconst.QuickCreateVCN] = &types.Flag{
		Type:  types.BoolType,
		Usage: "Quick Create VCN",
		Default: &types.Default{
			DefaultBool: false,
		},
	}

	driverFlag.Options[driverconst.CNIType] = &types.Flag{
		Type:  types.StringType,
		Usage: "Network type",
		Default: &types.Default{
			DefaultString: "FLANNEL_OVERLAY",
		},
	}

	driverFlag.Options[driverconst.KubernetesVersion] = &types.Flag{
		Type:  types.StringType,
		Usage: "The Kubernetes version that will be used for your master and worker nodes e.g. v1.25.4, v1.26.2",
		Default: &types.Default{
			DefaultString: "v1.25.4",
		},
	}

	driverFlag.Options[driverconst.NodePublicKeyContents] = &types.Flag{
		Type:  types.StringType,
		Usage: "The contents of the SSH public key to use for the nodes",
	}

	driverFlag.Options[driverconst.ImageDisplayName] = &types.Flag{
		Type:  types.StringType,
		Usage: "Image for cluster nodes",
	}
	driverFlag.Options[driverconst.CloudCredentialId] = &types.Flag{
		Type:  types.StringType,
		Usage: "The cloud credential id",
	}
	driverFlag.Options[driverconst.Region] = &types.Flag{
		Type:  types.StringType,
		Usage: "The cloud provider region",
	}
	driverFlag.Options[driverconst.CompartmentID] = &types.Flag{
		Type:  types.StringType,
		Usage: "The OCID of the compartment in which to create resources (VCN, worker nodes, etc.)",
	}
	driverFlag.Options[driverconst.VcnID] = &types.Flag{
		Type:  types.StringType,
		Usage: "The OCID of an existing virtual network to be used for cluster creation",
	}

	driverFlag.Options[driverconst.ImageId] = &types.Flag{
		Type:  types.StringType,
		Usage: "OCID for the node image (Optional)",
	}
	driverFlag.Options[driverconst.WorkerNodeSubnet] = &types.Flag{
		Type:  types.StringType,
		Usage: "OCID for node pool subnet",
	}
	driverFlag.Options[driverconst.ControlPlaneSubnet] = &types.Flag{
		Type:  types.StringType,
		Usage: "OCID for control plane subnet",
	}
	driverFlag.Options[driverconst.LoadBalancerSubnet] = &types.Flag{
		Type:  types.StringType,
		Usage: "OCID for load balancer subnet",
	}
	driverFlag.Options[driverconst.PodSubnet] = &types.Flag{
		Type:  types.StringType,
		Usage: "OCID for pod subnet",
	}

	driverFlag.Options[driverconst.RawNodePools] = &types.Flag{
		Type:  types.StringSliceType,
		Usage: "Cluster Node Pools",
		Default: &types.Default{
			DefaultStringSlice: &types.StringSlice{Value: []string{}}, // avoid nil value for init
		},
	}
	driverFlag.Options[driverconst.ApplyYAMLs] = &types.Flag{
		Type:  types.StringSliceType,
		Usage: "YAMLs to apply on managed cluster",
		Default: &types.Default{
			DefaultStringSlice: &types.StringSlice{Value: []string{}}, // avoid nil value for init
		},
	}
	driverFlag.Options[driverconst.VerrazzanoVersion] = &types.Flag{
		Type:  types.StringType,
		Usage: "The Verrazzano Version",
		Default: &types.Default{
			DefaultString: defaults.VerrazzanoVersion,
		},
	}
	driverFlag.Options[driverconst.VerrazzanoResource] = &types.Flag{
		Type:  types.StringType,
		Usage: "The Verrazzano resource to install on the managed cluster",
		Default: &types.Default{
			DefaultString: variables.DefaultVerrazzanoResource,
		},
	}
	driverFlag.Options[driverconst.InstallVerrazzano] = &types.Flag{
		Type:  types.BoolType,
		Usage: "Install Verrazzano addon",
		Default: &types.Default{
			DefaultBool: true,
		},
	}
	driverFlag.Options[driverconst.VerrazzanoTag] = &types.Flag{
		Type:  types.StringType,
		Usage: "The Verrazzano Tag Override",
	}
	driverFlag.Options[driverconst.CreateImagePullSecrets] = &types.Flag{
		Type:  types.BoolType,
		Usage: "Create the Verrazzano image pull secrets",
		Default: &types.Default{
			DefaultBool: false,
		},
	}
	driverFlag.Options[driverconst.ImagePullSecretUsername] = &types.Flag{
		Type:  types.StringType,
		Usage: "The image pull secret registry user name",
	}
	driverFlag.Options[driverconst.ImagePullSecretPassword] = &types.Flag{
		Type:  types.StringType,
		Usage: "The image pull secret registry user password",
	}
	driverFlag.Options[driverconst.ImagePullSecretEmail] = &types.Flag{
		Type:  types.StringType,
		Usage: "The image pull secret registry email",
	}
	d.Logger.Infof("capi.driver.GetDriverUpdateOptions(...) called returning driver flags %v", driverFlag)
	return &driverFlag, nil
}

// GetDriverUpdateOptions implements driver interface
func (d *OKEDriver) GetDriverUpdateOptions(ctx context.Context) (*types.DriverFlags, error) {
	d.Logger.Infof("capi.driver.GetDriverUpdateOptions(...) called")

	defaults, err := loadDefaults(ctx)
	if err != nil {
		return nil, err
	}

	driverFlag := types.DriverFlags{
		Options: make(map[string]*types.Flag),
	}

	driverFlag.Options[driverconst.ImageDisplayName] = &types.Flag{
		Type:  types.StringType,
		Usage: "Image for cluster nodes",
	}
	driverFlag.Options[driverconst.KubernetesVersion] = &types.Flag{
		Type:  types.StringType,
		Usage: "The Kubernetes version that will be used for your master and worker nodes e.g. v1.11.9, v1.12.7",
		Default: &types.Default{
			DefaultString: "v1.25.4",
		},
	}
	driverFlag.Options[driverconst.NodePublicKeyContents] = &types.Flag{
		Type:  types.StringType,
		Usage: "The contents of the SSH public key to use for the nodes",
	}
	driverFlag.Options[driverconst.RawNodePools] = &types.Flag{
		Type:  types.StringSliceType,
		Usage: "Cluster Node Pools",
		Default: &types.Default{
			DefaultStringSlice: &types.StringSlice{Value: []string{}}, // avoid nil value for init
		},
	}
	driverFlag.Options[driverconst.ApplyYAMLs] = &types.Flag{
		Type:  types.StringSliceType,
		Usage: "YAMLs to apply on managed cluster",
		Default: &types.Default{
			DefaultStringSlice: &types.StringSlice{Value: []string{}}, // avoid nil value for init
		},
	}
	driverFlag.Options[driverconst.VerrazzanoVersion] = &types.Flag{
		Type:  types.StringType,
		Usage: "The Verrazzano Version",
		Default: &types.Default{
			DefaultString: defaults.VerrazzanoVersion,
		},
	}
	driverFlag.Options[driverconst.VerrazzanoResource] = &types.Flag{
		Type:  types.StringType,
		Usage: "The Verrazzano resource to install on the managed cluster",
		Default: &types.Default{
			DefaultString: variables.DefaultVerrazzanoResource,
		},
	}
	driverFlag.Options[driverconst.InstallVerrazzano] = &types.Flag{
		Type:  types.BoolType,
		Usage: "Install Verrazzano addon",
		Default: &types.Default{
			DefaultBool: true,
		},
	}
	driverFlag.Options[driverconst.VerrazzanoTag] = &types.Flag{
		Type:  types.StringType,
		Usage: "The Verrazzano Tag Override",
	}
	driverFlag.Options[driverconst.ImagePullSecretUsername] = &types.Flag{
		Type:  types.StringType,
		Usage: "The image pull secret registry user name",
	}
	driverFlag.Options[driverconst.ImagePullSecretPassword] = &types.Flag{
		Type:  types.StringType,
		Usage: "The image pull secret registry user password",
	}
	driverFlag.Options[driverconst.ImagePullSecretEmail] = &types.Flag{
		Type:  types.StringType,
		Usage: "The image pull secret registry email",
	}
	return &driverFlag, nil
}

// Create implements driver interface
func (d *OKEDriver) Create(ctx context.Context, opts *types.DriverOptions, _ *types.ClusterInfo) (*types.ClusterInfo, error) {
	d.Logger.Infof("capi.driver.Create(...) called")
	vars, err := variables.NewFromOptions(ctx, opts)
	if err != nil {
		d.Logger.Errorf("error creating vars %v", err)
		return nil, err
	}
	plog, err := newProvisioningLogger(ctx, vars.DisplayName)
	if err != nil {
		return nil, err
	}
	_ = plog.Infof("Initializing cluster")
	/*
	* The ClusterInfo includes the following information Version, ServiceAccountToken,Endpoint, username, password, etc
	 */
	clusterInfo := &types.ClusterInfo{}
	err = storeVariables(clusterInfo, vars)
	if err != nil {
		d.Logger.Errorf("error storing vars %v", err)
		return clusterInfo, err
	}

	if err := d.doCreateOrUpdate(ctx, vars); err != nil {
		d.Logger.Errorf("Driver.Create: %v", err)
		return clusterInfo, err
	}
	return clusterInfo, nil
}

// Update implements driver interface
func (d *OKEDriver) Update(ctx context.Context, info *types.ClusterInfo, opts *types.DriverOptions) (*types.ClusterInfo, error) {
	d.Logger.Infof("capi.driver.Update(...) called")

	state, err := d.loadVariables(info)
	if err != nil {
		return info, err
	}
	newState, err := variables.NewFromOptions(ctx, opts)
	if err != nil {
		return info, err
	}

	if err := state.SetUpdateValues(ctx, newState); err != nil {
		return info, err
	}
	if err := storeVariables(info, state); err != nil {
		return info, err
	}
	di, err := k8s.InjectedDynamic()
	if err != nil {
		return info, err
	}
	ki, err := k8s.InjectedInterface()
	if err != nil {
		return info, err
	}

	plog := provisioning.NewLogger(ctx, ki, state.Name)
	if err := d.NewCAPIClient(plog).UpdateCluster(ctx, ki, di, state); err != nil {
		return info, err
	}

	return info, nil
}

func (d *OKEDriver) PostCheck(ctx context.Context, info *types.ClusterInfo) (*types.ClusterInfo, error) {
	d.Logger.Infof("capi.driver.PostCheck(...) called")

	state, err := d.loadVariables(info)
	if err != nil {
		return info, err
	}
	adminDi, err := k8s.InjectedDynamic()
	if err != nil {
		return info, err
	}
	adminKi, err := k8s.InjectedInterface()
	if err != nil {
		return info, err
	}
	plog := provisioning.NewLogger(ctx, adminKi, state.Name)
	if err := capi.IsCAPIClusterReady(ctx, adminDi, state, plog); err != nil {
		return info, err
	}
	capiClusterKubeConfig, err := state.GetCAPIClusterKubeConfig(ctx)
	if err != nil {
		return info, err
	}

	nc, err := state.NodeCount()
	if err != nil {
		return info, err
	}

	info.Version = state.KubernetesVersion
	info.Username = ""
	info.Password = ""
	info.ClientCertificate = ""
	info.ClientKey = ""
	info.NodeCount = nc.Count
	info.Metadata["nodePool"] = state.Name + "-1"
	if len(capiClusterKubeConfig.Clusters) > 0 {
		capiCluster := capiClusterKubeConfig.Clusters[0].Cluster
		info.Endpoint = capiCluster.Server
		info.RootCaCertificate = capiCluster.CertificateAuthorityData
	}

	// Use as a temporary token while we generate a service account.
	if len(capiClusterKubeConfig.Users) > 0 {
		if capiClusterKubeConfig.Users[0].User.Token != "" {
			info.ServiceAccountToken = capiClusterKubeConfig.Users[0].User.Token
		}
		// TODO handle info.ExecCredential when it is supported by Rancher
		// https://github.com/rancher/rancher/issues/24135
	}

	kubeConfigBytes, err := yaml.Marshal(&capiClusterKubeConfig)
	if err != nil {
		return info, fmt.Errorf("failed to get managed cluster kubeconfig: %v", err)
	}

	managedKI, err := k8s.NewInterfaceForKubeconfig(kubeConfigBytes)
	if err != nil {
		return info, fmt.Errorf("failed to create clientset for managed cluster %s: %v", state.Name, err)
	}

	d.Logger.Infof("Creating service account token for cluster %v", state.Name)
	hasToken := len(info.ServiceAccountToken) > 0
	info.ServiceAccountToken, err = d.generateServiceAccountToken(ctx, managedKI)
	if err != nil {
		return info, fmt.Errorf("could not generate service account token: %v", err)
	}
	// if we were able to generate the service account token for the first time, write a provisioning log message
	if !hasToken {
		_ = plog.Infof("Connected to cluster endpoint")
	}

	managedDI, err := k8s.NewDynamicForKubeconfig(kubeConfigBytes)
	if err != nil {
		return info, fmt.Errorf("failed to create dynamic clientset for managed cluster %s: %v", state.Name, err)
	}

	capiClient := d.NewCAPIClient(plog)
	if len(state.ApplyYAMLS) > 0 {
		d.Logger.Infof("Installing additional YAML documents on cluster %s", state.Name)
		if err := capiClient.CreateOrUpdateYAMLDocuments(ctx, managedDI, state); err != nil {
			return info, fmt.Errorf("failed to install additional YAML documents on cluster %s: %v", state.Name, err)
		}
	}
	if state.InstallVerrazzano {
		d.Logger.Infof("Updating Verrazzano on cluster %v", state.Name)
		if err := capiClient.UpdateVerrazzano(ctx, adminDi, state); err != nil {
			return info, err
		}
		// Create the image pull secret if required
		if err := capiClient.CreateImagePullSecrets(ctx, adminDi, state); err != nil {
			return info, err
		}
	} else {
		if err := capiClient.DeleteVerrazzanoResources(ctx, adminDi, state); err != nil {
			return info, err
		}
	}

	return info, nil
}

func (d *OKEDriver) GetClusterSize(_ context.Context, info *types.ClusterInfo) (*types.NodeCount, error) {
	v, err := d.loadVariables(info)
	if err != nil {
		return nil, err
	}
	return v.NodeCount()
}

func (d *OKEDriver) GetVersion(_ context.Context, info *types.ClusterInfo) (*types.KubernetesVersion, error) {
	v, err := d.loadVariables(info)
	if err != nil {
		return nil, err
	}
	return v.Version(), nil
}

func (d *OKEDriver) SetClusterSize(ctx context.Context, info *types.ClusterInfo, count *types.NodeCount) error {
	d.Logger.Infof("capi.driver.SetClusterSize(...) called")
	state, err := d.loadVariables(info)
	if err != nil {
		return err
	}

	if len(state.NodePools) > 0 {
		state.NodePools[0].Replicas = count.Count
	}
	if err := storeVariables(info, state); err != nil {
		d.Logger.Errorf("Failed to save new node group size: %v", err)
		return err
	}
	return d.doCreateOrUpdate(ctx, state)
}

// SetVersion sets the Kubernetes Version of cluster
func (d *OKEDriver) SetVersion(ctx context.Context, info *types.ClusterInfo, version *types.KubernetesVersion) error {
	d.Logger.Infof("capi.driver.SetVersion(...) called")
	state, err := d.loadVariables(info)
	if err != nil {
		return err
	}
	ki, err := k8s.InjectedInterface()
	if err != nil {
		return err
	}
	di, err := k8s.InjectedDynamic()
	if err != nil {
		return err
	}

	return d.NewCAPIClient(provisioning.NewLogger(ctx, ki, state.Name)).UpdateCluster(ctx, ki, di, state)
}

func (d *OKEDriver) GetCapabilities(_ context.Context) (*types.Capabilities, error) {
	d.Logger.Infof("capi.driver.GetCapabilities(...) called")
	return &d.driverCapabilities, nil
}

func (d *OKEDriver) ETCDSave(ctx context.Context, clusterInfo *types.ClusterInfo, opts *types.DriverOptions, snapshotName string) error {
	d.Logger.Infof("capi.driver.ETCDSave(...) called")
	return fmt.Errorf("ETCD backup operations are not implemented")
}

func (d *OKEDriver) ETCDRestore(ctx context.Context, clusterInfo *types.ClusterInfo, opts *types.DriverOptions, snapshotName string) (*types.ClusterInfo, error) {
	d.Logger.Infof("capi.driver.ETCDRestore(...) called")
	return nil, fmt.Errorf("ETCD backup operations are not implemented")
}

func (d *OKEDriver) ETCDRemoveSnapshot(ctx context.Context, clusterInfo *types.ClusterInfo, opts *types.DriverOptions, snapshotName string) error {
	d.Logger.Infof("capi.driver.ETCDRemoveSnapshot(...) called")
	return fmt.Errorf("ETCD backup operations are not implemented")
}

func (d *OKEDriver) GetK8SCapabilities(ctx context.Context, options *types.DriverOptions) (*types.K8SCapabilities, error) {
	d.Logger.Infof("capi.driver.GetK8SCapabilities(...) called")
	capabilities := &types.K8SCapabilities{
		L4LoadBalancer: &types.LoadBalancerCapabilities{
			Enabled:              true,
			Provider:             "OCILB",
			ProtocolsSupported:   []string{"TCP", "HTTP/1.0", "HTTP/1.1"},
			HealthCheckSupported: true,
		},
	}
	return capabilities, nil
}

func (d *OKEDriver) RemoveLegacyServiceAccount(ctx context.Context, info *types.ClusterInfo) error {
	d.Logger.Infof("capi.driver.RemoveLegacyServiceAccount(...) called")
	return nil
}

func storeVariables(info *types.ClusterInfo, v *variables.Variables) error {
	bytes, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("could not marshal state: %v", err)
	}

	if info.Metadata == nil {
		info.Metadata = map[string]string{}
	}

	info.Metadata[metadataKey] = string(bytes)
	return nil
}

func (d *OKEDriver) loadVariables(info *types.ClusterInfo) (*variables.Variables, error) {
	d.Logger.Infof("capi.driver.loadVariables(...) called")
	state := &variables.Variables{}
	err := json.Unmarshal([]byte(info.Metadata[metadataKey]), &state)
	return state, err
}

// GenerateServiceAccountToken generate a serviceAccountToken for clusterAdmin given a clientset
func (d *OKEDriver) generateServiceAccountToken(ctx context.Context, clientset kubernetes.Interface) (string, error) {

	token := ""
	namespace := "default"
	name := "kontainer-engine-oke-capi"

	serverVersion, err := clientset.Discovery().ServerVersion()
	if err != nil {
		return "", err
	}
	d.Logger.Debugf("[oraclecontainerengine] Kubernetes server version: %s", serverVersion)

	serviceAccount := &v1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: name}}

	// Create new service account, if it does not exist already
	_, err = clientset.CoreV1().ServiceAccounts(namespace).Create(ctx, serviceAccount, metav1.CreateOptions{})
	if err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return "", err
		}
	}

	serviceAccount, err = clientset.CoreV1().ServiceAccounts(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	// Template for an authentication token secret bound to the service account
	secretTemplate := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: serviceAccount.Name + "-token",
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "v1",
					Kind:       "ServiceAccount",
					Name:       serviceAccount.Name,
					UID:        serviceAccount.UID,
				},
			},
			Annotations: map[string]string{
				v1.ServiceAccountNameKey: serviceAccount.Name,
			},
		},
		Type: v1.SecretTypeServiceAccountToken,
	}

	_, err = clientset.CoreV1().Secrets(namespace).Create(ctx, secretTemplate, metav1.CreateOptions{})
	if err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return "", err
		}
	}
	// wait a few seconds for authentication token to populate
	time.Sleep(10 * time.Second)

	secretObj, err := clientset.CoreV1().Secrets(namespace).Get(ctx, serviceAccount.Name+"-token", metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	// Create new cluster-role-bindings, if it does not exist already
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Subjects: []rbacv1.Subject{{Kind: rbacv1.ServiceAccountKind, APIGroup: "", Name: name, Namespace: namespace}},
		RoleRef: rbacv1.RoleRef{
			Kind: "ClusterRole",
			Name: "cluster-admin",
		},
	}

	_, err = clientset.RbacV1().ClusterRoleBindings().Create(ctx, clusterRoleBinding, metav1.CreateOptions{})
	if err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return "", err
		}
	}

	// Look up cluster role binding
	_, err = clientset.RbacV1().ClusterRoleBindings().Get(ctx, clusterRoleBinding.Name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("error getting cluster role binding: %v", err)
	}

	// get the bearer token from the token-secret
	if byteToken, ok := secretObj.Data[v1.ServiceAccountTokenKey]; ok {
		token = string(byteToken)
		return token, nil
	}

	return "", fmt.Errorf("error getting authentication token from secret: %s", secretObj.Name)
}

func (d *OKEDriver) doCreateOrUpdate(ctx context.Context, state *variables.Variables) error {
	dynamicInterface, err := k8s.InjectedDynamic()
	if err != nil {
		return fmt.Errorf("failed to get dynamicInterface: %v", err)
	}
	kubernetesInterface, err := k8s.InjectedInterface()
	if err != nil {
		return fmt.Errorf("failed to get kubernetesInterface: %v", err)
	}
	_, err = d.NewCAPIClient(provisioning.NewLogger(ctx, kubernetesInterface, state.Name)).CreateOrUpdateAllObjects(ctx, kubernetesInterface, dynamicInterface, state)
	if err != nil {
		return fmt.Errorf("failed to create objects: %v", err)
	}
	return nil
}

func loadDefaults(ctx context.Context) (*version.Defaults, error) {
	ki, err := k8s.InjectedInterface()
	if err != nil {
		return nil, fmt.Errorf("failed to load kubernetes interface for default values: %v", err)
	}
	defaults, err := version.LoadDefaults(ctx, ki)
	if err != nil {
		return nil, fmt.Errorf("failed to load default values: %v", err)
	}
	return defaults, nil
}

func (d *OKEDriver) NewCAPIClient(logger *provisioning.Logger) *capi.CAPIClient {
	return capi.NewCAPIClient(logger)
}

func newProvisioningLogger(ctx context.Context, name string) (*provisioning.Logger, error) {
	ki, err := k8s.InjectedInterface()
	if err != nil {
		return nil, err
	}
	return provisioning.NewLogger(ctx, ki, name), nil
}
