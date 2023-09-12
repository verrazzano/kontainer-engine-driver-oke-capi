// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package variables

import (
	"crypto/sha256"
	"encoding/base32"
	"fmt"
	"strings"
)

/*
 * Compute hashes for Control plane and node pools.
 * Hashes are used to determine if there is a change in the node pool, and trigger rolling upgrades.
 */

func (v *Variables) SetHashes() {
	v.SetControlPlaneHash()
	v.SetNodePoolHash()
}

func (v *Variables) SetControlPlaneHash() {
	b := strings.Builder{}
	b.WriteString(v.KubernetesVersion)
	b.WriteString(v.SSHPublicKey)
	v.ControlPlaneHash = hashSum(b.String())
}

func (v *Variables) SetNodePoolHash() {
	b := strings.Builder{}
	b.WriteString(v.KubernetesVersion)
	b.WriteString(v.SSHPublicKey)
	b.WriteString(v.ActualImage)
	// changing node pool replicas does not require a new template hash, since it is a scale up/scale down
	for _, np := range v.NodePools {
		b.WriteString(np.Name)
		b.WriteString(np.Shape)
		b.WriteString(fmt.Sprintf("%d", np.Memory))
		b.WriteString(fmt.Sprintf("%d", np.Ocpus))
	}
	v.NodePoolHash = hashSum(b.String())
}

func hashSum(input string) string {
	sha := sha256.New()
	sha.Write([]byte(input))
	encoded := base32.StdEncoding.EncodeToString(sha.Sum(nil))
	return strings.ToLower(encoded[0:5])
}
