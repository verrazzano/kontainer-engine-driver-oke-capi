// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package provisioning

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/kontainer-engine-driver-oke-capi/pkg/capi/object"
	"github.com/verrazzano/kontainer-engine-driver-oke-capi/pkg/variables"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"testing"
)

const (
	testClusterName = "kluster"
	testMsg1        = "first message"
	testMsg2        = "hi %s"

	testClusterObject = `apiVersion: cluster.x-k8s.io/v1beta1
kind: Cluster
metadata:
  creationTimestamp: "2023-07-18T20:07:23Z"
  finalizers:
  - cluster.cluster.x-k8s.io
  generation: 17
  labels:
    cluster.x-k8s.io/cluster-name: c-2jcfm
  name: c-2jcfm
  namespace: c-2jcfm
  resourceVersion: "26449824"
  uid: f7a987e0-398a-4959-9590-009217084f7e
spec:
  clusterNetwork:
    pods:
      cidrBlocks:
      - 10.244.0.0/16
    serviceDomain: cluster.local
    services:
      cidrBlocks:
      - 10.96.0.0/16
  controlPlaneEndpoint:
    host: 10.196.5.74
    port: 6443
  controlPlaneRef:
    apiVersion: controlplane.cluster.x-k8s.io/v1beta2
    kind: OCIManagedControlPlane
    name: c-2jcfm-control-plane
    namespace: c-2jcfm
  infrastructureRef:
    apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
    kind: OCICluster
    name: c-2jcfm
    namespace: c-2jcfm
status:
  conditions:
  - lastTransitionTime: "2023-07-18T20:46:33Z"
    message: Scaling up control plane to 1 replicas (actual 0)
    reason: ScalingUp
    severity: Warning
    status: "False"
    type: Ready
  - lastTransitionTime: "2023-07-18T20:07:24Z"
    message: Waiting for control plane provider to indicate the control plane has
      been initialized
    reason: WaitingForControlPlaneProviderInitialized
    severity: Info
    status: "False"
    type: ControlPlaneInitialized
  - lastTransitionTime: "2023-07-18T20:07:24Z"
    message: Waiting for OCI instance
    reason: ScalingUp
    severity: Warning
    status: "False"
    type: ControlPlaneReady
  - lastTransitionTime: "2023-07-18T20:46:33Z"
    status: "True"
    type: InfrastructureReady
  failureDomains:
    "1":
      attributes:
        AvailabilityDomain: YRCi:US-ASHBURN-AD-1
      controlPlane: true
    "2":
      attributes:
        AvailabilityDomain: YRCi:US-ASHBURN-AD-2
      controlPlane: true
    "3":
      attributes:
        AvailabilityDomain: YRCi:US-ASHBURN-AD-3
      controlPlane: true
  infrastructureReady: true
  observedGeneration: 17
  phase: Provisioned`
)

func TestLogger(t *testing.T) {
	ki := fake.NewSimpleClientset()
	ctx := context.TODO()
	log := NewLogger(ctx, ki, testClusterName)
	err := log.Infof(testMsg1)
	assert.NoError(t, err)
	// write one message
	_ = assertLastMessage(t, ctx, ki, testMsg1)
	err = log.Errorf(testMsg2, "from test cluster")
	assert.NoError(t, err)
	// write another message. It should be the last message
	_ = assertLastMessage(t, ctx, ki, "hi from test cluster")
	longMsg := makeLongTestString('a', maxLength)
	err = log.Infof(longMsg)
	assert.NoError(t, err)
	_ = assertLastMessage(t, ctx, ki, longMsg)
	_ = log.Infof(testMsg1)
	p := assertLastMessage(t, ctx, ki, testMsg1)
	// log should be truncated at this point
	assert.Less(t, len(p), maxLength)
}

func TestClusterStatus(t *testing.T) {
	ki := fake.NewSimpleClientset()
	ctx := context.TODO()
	log := NewLogger(ctx, ki, testClusterName)
	cl, err := object.LoadTextTemplate(object.Object{
		Text: testClusterObject,
	}, variables.Variables{})
	assert.NoError(t, err)
	_ = log.ClusterStatus(&cl[0])
	_ = assertLastMessage(t, ctx, ki, "Waiting for control plane provider to indicate the control plane has been initialized: Scaling up control plane to 1 replicas (actual 0), Waiting for OCI instance")
}

func assertLastMessage(t *testing.T, ctx context.Context, ki kubernetes.Interface, msg string) string {
	cm, err := ki.CoreV1().ConfigMaps(testClusterName).Get(ctx, configMapName, metav1.GetOptions{})
	assert.NoError(t, err)
	if cm == nil {
		return ""
	}
	last := cm.Data[lastLogField]
	assert.Contains(t, last, msg)
	return cm.Data[logField]
}

func makeLongTestString(r byte, length int) string {
	b := make([]byte, length)
	for i := 0; i < length; i++ {
		b[i] = r
	}
	return string(b)
}
