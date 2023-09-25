// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capi

import (
	"context"
	"github.com/stretchr/testify/assert"
	fakelogger "github.com/verrazzano/kontainer-engine-driver-oke-capi/pkg/provisioning/fake"
	"github.com/verrazzano/kontainer-engine-driver-oke-capi/pkg/variables"
	"k8s.io/apimachinery/pkg/runtime"
	fake2 "k8s.io/client-go/dynamic/fake"
	"testing"
)

func TestUpdateVerrazzano(t *testing.T) {
	v := &variables.Variables{
		Name:               testName,
		Namespace:          testName,
		InstallVerrazzano:  true,
		VerrazzanoResource: variables.DefaultVerrazzanoResource,
	}

	c := NewCAPIClient(fakelogger.NewLogger())
	scheme := runtime.NewScheme()
	adminDi := fake2.NewSimpleDynamicClient(scheme)
	err := c.UpdateVerrazzano(context.TODO(), adminDi, v)
	assert.NoError(t, err)
}