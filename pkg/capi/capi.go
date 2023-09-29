// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capi

import (
	"context"
	"errors"
	"fmt"
	"github.com/verrazzano/kontainer-engine-driver-oke-capi/pkg/capi/object"
	"github.com/verrazzano/kontainer-engine-driver-oke-capi/pkg/gvr"
	"github.com/verrazzano/kontainer-engine-driver-oke-capi/pkg/provisioning"
	"github.com/verrazzano/kontainer-engine-driver-oke-capi/pkg/templates"
	"github.com/verrazzano/kontainer-engine-driver-oke-capi/pkg/variables"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"strings"
	"time"
)

const (
	ociTenancyField              = "tenancy"
	ociUserField                 = "user"
	ociFingerprintField          = "fingerprint"
	ociRegionField               = "region"
	ociPassphraseField           = "passphrase"
	ociKeyField                  = "key"
	ociUseInstancePrincipalField = "useInstancePrincipal"
)

const (
	clusterPhaseProvisioned = "Provisioned"
	machinePoolPhaseRunning = "Running"
)

type CAPIClient struct {
	verrazzanoTimeout         time.Duration
	verrazzanoPollingInterval time.Duration
	plog                      *provisioning.Logger
}

func NewCAPIClient(plog *provisioning.Logger) *CAPIClient {
	return &CAPIClient{
		verrazzanoTimeout:         5 * time.Minute,
		verrazzanoPollingInterval: 10 * time.Second,
		plog:                      plog,
	}
}

func (c *CAPIClient) DeleteHangingResources(ctx context.Context, p dynamic.Interface, v *variables.Variables) error {
	return deleteWorkerObjects(ctx, p, v.Namespace, v)
}

func (c *CAPIClient) CreateOrUpdateYAMLDocuments(ctx context.Context, managedDi dynamic.Interface, v *variables.Variables) error {
	_, err := createOrUpdateObjects(ctx, managedDi, object.ToObjects(v.ApplyYAMLS), v)
	return err
}

// CreateOrUpdateAllObjects creates or updates all cluster result
func (c *CAPIClient) CreateOrUpdateAllObjects(ctx context.Context, kubernetesInterface kubernetes.Interface, dynamicInterface dynamic.Interface, v *variables.Variables) (*CreateOrUpdateResult, error) {
	if err := createOrUpdateCAPISecret(ctx, v, kubernetesInterface); err != nil {
		return nil, fmt.Errorf("failed to create CAPI credentials: %v", err)
	}
	return createOrUpdateObjects(ctx, dynamicInterface, object.CreateObjects(), v)
}

// createOrUpdateCAPISecret creates the CAPI secret if it does not already exist
// if the secret exists, update it in place with the new credentials
func createOrUpdateCAPISecret(ctx context.Context, v *variables.Variables, client kubernetes.Interface) error {
	data := map[string][]byte{
		ociTenancyField:              []byte(v.Tenancy),
		ociUserField:                 []byte(v.User),
		ociFingerprintField:          []byte(v.Fingerprint),
		ociRegionField:               []byte(v.Region),
		ociPassphraseField:           []byte(v.PrivateKeyPassphrase),
		ociKeyField:                  []byte(strings.TrimSpace(v.PrivateKey)),
		ociUseInstancePrincipalField: []byte("false"),
	}
	secretName := fmt.Sprintf("%s-principal", v.Name)
	current, err := client.CoreV1().Secrets(v.Namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		// Create if not exists
		if apierrors.IsNotFound(err) {
			_, err := client.CoreV1().Secrets(v.Namespace).Create(ctx, &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: secretName,
					Labels: map[string]string{
						"cluster.x-k8s.io/provider": "infrastructure-oci",
					},
				},
				Data: data,
			}, metav1.CreateOptions{})
			return err
		}
		return err
	}

	// update secret in place
	current.Data = data
	_, err = client.CoreV1().Secrets(v.Namespace).Update(ctx, current, metav1.UpdateOptions{})
	return err
}

func createOrUpdateObjects(ctx context.Context, dynamicInterface dynamic.Interface, objects []object.Object, v *variables.Variables) (*CreateOrUpdateResult, error) {
	cruResult := NewCreateOrUpdateResult()
	for _, o := range objects {
		partialResult, err := createOrUpdateObject(ctx, dynamicInterface, o, v)
		if err != nil {
			return cruResult, fmt.Errorf("object processing error: %v", err)
		}
		cruResult.Merge(partialResult)
	}
	return cruResult, nil
}

func createOrUpdateObject(ctx context.Context, client dynamic.Interface, o object.Object, v *variables.Variables) (*CreateOrUpdateResult, error) {
	return cruObject(ctx, client, o, v, func(u *unstructured.Unstructured) error { return nil })
}

// cruObject create or update an object
func cruObject(ctx context.Context, client dynamic.Interface, o object.Object, v *variables.Variables, updater func(u *unstructured.Unstructured) error) (*CreateOrUpdateResult, error) {
	cruResult := NewCreateOrUpdateResult()
	toCreateObject, err := object.LoadTextTemplate(o, *v)
	if err != nil {
		return cruResult, err
	}

	for idx := range toCreateObject {
		u := &toCreateObject[idx]
		// Try to fetch existing object
		groupVersionResource := object.GVR(u)
		existingObject, err := client.Resource(groupVersionResource).Namespace(object.DefaultingNamespace(u)).Get(ctx, u.GetName(), metav1.GetOptions{})
		if err != nil {
			// if object doesn't exist, try to create it
			if apierrors.IsNotFound(err) {
				if err := createIfNotExists(ctx, client, u); err != nil {
					return cruResult, fmt.Errorf("create failed %s/%s/%s: %v", groupVersionResource.Group, groupVersionResource.Version, groupVersionResource.Resource, err)
				}
			} else {
				return cruResult, fmt.Errorf("get failed %s/%s/%s: %v", groupVersionResource.Group, groupVersionResource.Version, groupVersionResource.Resource, err)
			}
		} else { // If the Object exists, merge with existingObject and do an update
			mergedObject := mergeUnstructured(existingObject, u, o.LockedFields)
			if err != nil {
				return cruResult, fmt.Errorf("merge failed %s/%s/%s: %v", groupVersionResource.Group, groupVersionResource.Version, groupVersionResource.Resource, err)
			}
			if err := updater(mergedObject); err != nil {
				return cruResult, fmt.Errorf("spec update failed %s/%s/%s: %v", groupVersionResource.Group, groupVersionResource.Version, groupVersionResource.Resource, err)
			}
			_, err = client.Resource(groupVersionResource).Namespace(object.DefaultingNamespace(mergedObject)).Update(context.TODO(), mergedObject, metav1.UpdateOptions{})
			if err != nil {
				return cruResult, fmt.Errorf("update failed %s/%s/%s: %v", groupVersionResource.Group, groupVersionResource.Version, groupVersionResource.Resource, err)
			}
		}

		cruResult.Add(groupVersionResource.Resource, u)
	}

	return cruResult, nil
}

func createIfNotExists(ctx context.Context, client dynamic.Interface, u *unstructured.Unstructured) error {
	_, err := client.Resource(object.GVR(u)).Namespace(object.DefaultingNamespace(u)).Create(ctx, u, metav1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		return nil
	}
	return err
}

// DeleteCluster deletes the cluster
func (c *CAPIClient) DeleteCluster(ctx context.Context, di dynamic.Interface, ki kubernetes.Interface, v *variables.Variables) error {
	clusterTmpl := object.Object{
		Text: templates.Cluster,
	}
	us, err := object.LoadTextTemplate(clusterTmpl, *v)
	// This error should only be hit if the Cluster YAML is syntactically incorrect, which is unlikely to happen.
	if err != nil {
		return errors.New("failed to delete cluster, remove cluster resources manually")
	}
	// There should always be exactly one cluster resource. If this is not the case, manual cleanup is necessary.
	if len(us) != 1 {
		return errors.New("invalid cluster, remove cluster resources manually")
	}
	cluster := &us[0]
	clusterGVR := object.GVR(cluster)

	get, err := di.Resource(clusterGVR).Namespace(object.DefaultingNamespace(cluster)).Get(ctx, cluster.GetName(), metav1.GetOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		// This is most likely to be a transient or environmental error with the cluster
		return errors.New("failed to lookup cluster during delete")
	}

	// If no cluster, delete the cluster namespace
	if get == nil {
		err := ki.CoreV1().Namespaces().Delete(ctx, cluster.GetName(), metav1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return errors.New("failed to delete cluster namespace")
		}
		return nil
	}

	// Delete the cluster if not already being deleted
	if get.GetDeletionTimestamp() == nil {
		err := di.Resource(clusterGVR).Namespace(object.DefaultingNamespace(cluster)).Delete(ctx, cluster.GetName(), metav1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return errors.New("failed to delete cluster")
		}
	}
	_ = c.plog.Infof("Deleting cluster")
	// Surface that the cluster is being deleted to the user
	return errors.New("deleting cluster")
}

func deleteUnstructureds(ctx context.Context, di dynamic.Interface, us []unstructured.Unstructured) error {
	for idx := range us {
		u := &us[idx]
		groupVersionResource := object.GVR(u)
		return deleteIfExists(ctx, di, groupVersionResource, u.GetName(), object.DefaultingNamespace(u))
	}
	return nil
}

func deleteIfExists(ctx context.Context, di dynamic.Interface, gvr schema.GroupVersionResource, name, namespace string) error {
	err := di.Resource(gvr).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}

func deleteWorkerObjects(ctx context.Context, di dynamic.Interface, namespace string, v *variables.Variables) error {
	fieldSelector := fmt.Sprintf("metadata.namespace=%s", namespace)
	// cleanup machine pools
	mps, err := di.Resource(gvr.MachinePool).List(ctx, metav1.ListOptions{
		FieldSelector: fieldSelector,
	})
	if err != nil {
		return err
	}

	// Delete unused machinepools
	for _, mp := range mps.Items {
		// delete any machine pools that were not in the CRU
		deleted, err := deleteIfNotCRU(ctx, di, v, &mp)
		if err != nil {
			return err
		}
		if deleted {
			// delete associated ocimachinepool if it exists
			poolName, err := object.NestedField(mp.Object, "spec", "template", "spec", "infrastructureRef", "name")
			if ociMachinePool, ok := poolName.(string); ok && err == nil {
				if err := deleteIfExists(ctx, di, gvr.OCIMachinePools, ociMachinePool, namespace); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func deleteIfNotCRU(ctx context.Context, di dynamic.Interface, v *variables.Variables, u *unstructured.Unstructured) (bool, error) {
	for _, np := range v.NodePools {
		if np.Name == u.GetName() {
			return false, nil
		}
	}
	return true, deleteUnstructureds(ctx, di, []unstructured.Unstructured{*u})
}
