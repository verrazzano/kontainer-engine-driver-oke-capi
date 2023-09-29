// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capi

import (
	"context"
	"errors"
	"fmt"
	"github.com/verrazzano/kontainer-engine-driver-oke-capi/pkg/capi/object"
	"github.com/verrazzano/kontainer-engine-driver-oke-capi/pkg/gvr"
	"github.com/verrazzano/kontainer-engine-driver-oke-capi/pkg/templates"
	"github.com/verrazzano/kontainer-engine-driver-oke-capi/pkg/variables"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
)

const (
	verrazzanoMCNamespace = "verrazzano-mc"
)

func (c *CAPIClient) UpdateVerrazzano(ctx context.Context, adminDi dynamic.Interface, v *variables.Variables) error {
	if !v.InstallVerrazzano || v.VerrazzanoResource == "" {
		return nil
	}
	// Create the Verrazzano Fleet Resource
	if err := createOrUpdateVerrazzano(ctx, adminDi, v); err != nil {
		_ = c.plog.Errorf("Failed to install Verrazzano")
		return fmt.Errorf("verrazzano install/update error: %v", err)
	}
	return c.plog.Infof("Updated Verrazzano")
}

// DeleteVerrazzanoResources deletes the Verrazzano resource on the managed cluster, and the VerrazzanoManagedCluster on the admin cluster
func (c *CAPIClient) DeleteVerrazzanoResources(ctx context.Context, adminDi dynamic.Interface, v *variables.Variables) error {
	if v.UninstallVerrazzano || v.InstallVerrazzano {
		_ = c.plog.Infof("Uninstalling Verrazzano on cluster %v", v.Name)
		if err := deleteVMC(ctx, adminDi, v); err != nil {
			return err
		}
		if err := c.deleteVZ(ctx, adminDi, v); err != nil {
			return err
		}
	}
	return nil
}

func (c *CAPIClient) CreateImagePullSecrets(ctx context.Context, adminDi dynamic.Interface, v *variables.Variables) error {
	if v.CreateImagePullSecrets {
		if _, err := createOrUpdateObject(ctx, adminDi, object.Object{
			Text: templates.ImagePullSecret,
		}, v); err != nil {
			return fmt.Errorf("image pull secret(s) creation error: %v", err)
		}
	}
	return nil
}

func deleteVMC(ctx context.Context, adminDi dynamic.Interface, v *variables.Variables) error {
	// Clean up the admin cluster VMC
	err := adminDi.Resource(gvr.VerrazzanoManagedCluster).Namespace(verrazzanoMCNamespace).Delete(ctx, v.Name, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) && !meta.IsNoMatchError(err) {
		// IsNoMatchError ignored in case cluster-operator not installed, and the VMC CRD is not present
		return fmt.Errorf("failed to delete Verrazzano Managed cluster: %v", err)
	}

	return nil
}

func (c *CAPIClient) deleteVZ(ctx context.Context, adminDi dynamic.Interface, v *variables.Variables) error {
	vzFleet, err := getVerrazzanoFleet(v)
	if err != nil {
		return err
	}
	err = adminDi.Resource(gvr.VerrazzanoFleet).Namespace(object.DefaultingNamespace(vzFleet)).Delete(ctx, vzFleet.GetName(), metav1.DeleteOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to delete Verrazzano resource: %v", err)
	}

	_ = c.plog.Infof("Uninstalling Verrazzano")
	return errors.New("uninstalling Verrazzano")
}

func getVerrazzanoFleet(v *variables.Variables) (*unstructured.Unstructured, error) {
	// Load the VZ from template and clean the managed cluster VZ
	us, err := object.LoadTextTemplate(object.Object{
		Text: templates.VerrazzanoFleet,
	}, *v)
	if err != nil {
		return nil, err
	}
	if len(us) != 1 {
		return nil, fmt.Errorf("expected 1 Verrazzano resource from template, got %d", len(us))
	}
	vzFleet := &us[0]
	return vzFleet, nil
}

func createOrUpdateVerrazzano(ctx context.Context, di dynamic.Interface, v *variables.Variables) error {
	if _, err := cruObject(ctx, di, object.Object{
		Text: templates.VerrazzanoFleet,
	}, v, func(u *unstructured.Unstructured) error {
		return unstructured.SetNestedField(u.Object, v.VerrazzanoVersion, "spec", "verrazzano", "spec", "version")
	}); err != nil {
		return err
	}

	return nil
}
