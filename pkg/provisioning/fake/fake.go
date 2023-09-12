// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package fake

import (
	"context"
	"github.com/verrazzano/kontainer-engine-driver-oke-capi/pkg/provisioning"
	"k8s.io/client-go/kubernetes/fake"
)

func NewLogger() *provisioning.Logger {
	return provisioning.NewLogger(context.TODO(), fake.NewSimpleClientset(), "fake")
}
