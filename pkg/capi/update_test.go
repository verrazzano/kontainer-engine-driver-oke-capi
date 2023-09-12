// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capi

import (
	"context"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes/fake"
	"testing"
)

func TestUpdateCluster(t *testing.T) {
	di := createTestDIWithClusterAndMachine()
	ki := fake.NewSimpleClientset()
	err := testCAPIClient.UpdateCluster(context.TODO(), ki, di, testVariables)
	assert.NoError(t, err)
}
