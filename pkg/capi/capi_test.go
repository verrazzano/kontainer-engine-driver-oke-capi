// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capi

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/kontainer-engine-driver-oke-capi/pkg/capi/object"
	"github.com/verrazzano/kontainer-engine-driver-oke-capi/pkg/gvr"
	fakelogger "github.com/verrazzano/kontainer-engine-driver-oke-capi/pkg/provisioning/fake"
	"github.com/verrazzano/kontainer-engine-driver-oke-capi/pkg/variables"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	fake2 "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
)

const (
	testName = "test"

	testMachine = `apiVersion: cluster.x-k8s.io/v1beta1
kind: MachinePool
metadata:
  labels:
    cluster.x-k8s.io/cluster-name: test
    cluster.x-k8s.io/control-plane: ""
    cluster.x-k8s.io/control-plane-name: test
  namespace: test`
	testKey = `aaa
bbb
ccc
ddd`
)

var (
	testCAPIClient = &CAPIClient{}

	testVariables = &variables.Variables{
		Name:              testName,
		Namespace:         testName,
		CloudCredentialId: "cattle-global-data:admin-creds",
		Tenancy:           "t",
		User:              "u",
		PrivateKey:        "k",
		Region:            "r",
	}
)

func TestNewCAPIClient(t *testing.T) {
	c := NewCAPIClient(fakelogger.NewLogger())
	z := 0 * time.Second
	assert.Greater(t, c.verrazzanoTimeout, z)
	assert.Greater(t, c.verrazzanoPollingInterval, z)
}

func TestCreateOrUpdateAllObjects(t *testing.T) {
	ki := fake.NewSimpleClientset()
	di := createTestDIWithClusterAndMachine()
	_, err := testCAPIClient.CreateOrUpdateAllObjects(context.TODO(), ki, di, testVariables)
	assert.NoError(t, err)
}

func TestRenderObjects(t *testing.T) {
	v := variables.Variables{
		DisplayName:          "xyz",
		Name:                 "xyz",
		Namespace:            "xyz",
		CompartmentID:        "ocid1.compartment.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ImageID:              "ocid1.image.oc1.iad.aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		VCNID:                "ocid1.vcn.oc1.iad.aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		WorkerNodeSubnet:     "ocid1.subnet.oc1.iad.aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ControlPlaneSubnet:   "ocid1.subnet.oc1.iad.yyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy",
		LoadBalancerSubnet:   "ocid1.subnet.oc1.iad.xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
		SSHPublicKey:         "ssh-rsa aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa foo@foo-mac",
		KubernetesVersion:    "v1.24.8",
		PodCIDR:              "192.168.0.0/16",
		ClusterCIDR:          "10.0.0.0/12",
		Fingerprint:          "fingerprint",
		PrivateKey:           testKey,
		PrivateKeyPassphrase: "",
		Region:               "xyz",
		Tenancy:              "xyz",
		User:                 "xyz",
		Hash:                 "xyz",
		ProviderId:           variables.ProviderId,

		NodePools: []variables.NodePool{
			{
				Name:       "np-1",
				Replicas:   1,
				Memory:     32,
				Ocpus:      4,
				VolumeSize: 100,
				Shape:      "VM.E4.Standard.Flex",
			},
			{
				Name:       "np-2",
				Replicas:   2,
				Memory:     64,
				Ocpus:      8,
				VolumeSize: 250,
				Shape:      "xyz",
			},
		},
	}

	os := object.CreateObjects()
	for _, o := range os {
		u, err := object.LoadTextTemplate(o, v)
		assert.NoError(t, err)
		assert.NotNil(t, u)
	}
}

func TestDeleteCluster(t *testing.T) {
	cluster := createTestCluster(testVariables, true, true, clusterPhaseProvisioned)
	ki := fake.NewSimpleClientset()
	var tests = []struct {
		name     string
		di       dynamic.Interface
		deleting bool
	}{
		{
			"delete no cluster",
			fake2.NewSimpleDynamicClient(runtime.NewScheme()),
			false,
		},
		{
			"delete with cluster",
			fake2.NewSimpleDynamicClient(runtime.NewScheme(), cluster),
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewCAPIClient(fakelogger.NewLogger()).DeleteCluster(context.TODO(), tt.di, ki, testVariables)
			if tt.deleting {
				if err == nil {
					assert.Fail(t, "expected progress message")
				}
				assert.Equal(t, "deleting cluster", err.Error())
			}
		})
	}
}

func listGVK(u *unstructured.Unstructured) schema.GroupVersionKind {
	gvk := u.GroupVersionKind()
	gvk.Kind = gvk.Kind + "List"
	return gvk
}

func createTestMachine(v *variables.Variables, phase string) *unstructured.Unstructured {
	machine, err := object.LoadTextTemplate(object.Object{
		Text: testMachine,
	}, *v)
	if err != nil {
		panic(err)
	}
	machine[0].Object["status"] = map[string]interface{}{
		"phase": phase,
	}
	machine[0].Object["metadata"].(map[string]interface{})["name"] = fmt.Sprintf("m-%d", rand.Intn(10000))
	return &machine[0]
}

func createTestCluster(v *variables.Variables, cReady, iReady bool, phase string) *unstructured.Unstructured {
	cluster, err := object.LoadTextTemplate(object.CAPICluster, *v)
	if err != nil {
		panic(err)
	}
	cluster[0].Object["status"] = map[string]interface{}{
		"controlPlaneReady":   cReady,
		"infrastructureReady": iReady,
		"phase":               phase,
	}
	return &cluster[0]
}

func createTestDIWithClusterAndMachine() dynamic.Interface {
	cluster := createTestCluster(testVariables, true, true, clusterPhaseProvisioned)
	machine := createTestMachine(testVariables, machinePhaseRunning)
	scheme := runtime.NewScheme()
	scheme.AddKnownTypeWithName(listGVK(machine), &unstructured.UnstructuredList{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   gvr.MachinePool.Group,
		Version: gvr.MachinePool.Version,
		Kind:    "MachinePoolList",
	}, &unstructured.UnstructuredList{})
	di := fake2.NewSimpleDynamicClient(scheme, cluster, machine)
	return di
}
