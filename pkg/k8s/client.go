// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8s

import (
	"encoding/base64"
	"errors"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"os"
)

const (
	injectedKubeConfig = "INJECTED_KUBECONFIG"
)

var (
	kubernetesInterface kubernetes.Interface
	dynamicInterface    dynamic.Interface
)

// MustSetKubeconfigFromEnv sets the current kubeconfig from the environment. If the kubeconfig cannot be set, panic.
func MustSetKubeconfigFromEnv() {
	val := os.Getenv(injectedKubeConfig)

	if len(val) < 1 {
		panic(errors.New("injected KubeConfig not found"))
	}

	kc, err := base64.StdEncoding.DecodeString(val)
	if err != nil {
		panic(err)
	}

	InjectedKubeConfig = kc
}

var InjectedKubeConfig []byte

// NewInterfaceForKubeconfig creates a kubernetes.Interface given a kubeconfig string
func NewInterfaceForKubeconfig(kubeconfig []byte) (kubernetes.Interface, error) {
	config, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

func NewDynamicForKubeconfig(kubeconfig []byte) (dynamic.Interface, error) {
	config, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
	if err != nil {
		return nil, err
	}
	return dynamic.NewForConfig(config)
}

// InjectedInterface creates a new kubernetes.Interface using the injected kubeconfig
func InjectedInterface() (kubernetes.Interface, error) {
	if kubernetesInterface != nil {
		return kubernetesInterface, nil
	}
	ki, err := NewInterfaceForKubeconfig(InjectedKubeConfig)
	if err != nil {
		return nil, err
	}
	kubernetesInterface = ki
	return kubernetesInterface, nil

}

// InjectedDynamic creates a new dynamic.Interface using the injected kubeconfig
func InjectedDynamic() (dynamic.Interface, error) {
	if dynamicInterface != nil {
		return dynamicInterface, nil
	}
	di, err := NewDynamicForKubeconfig(InjectedKubeConfig)
	if err != nil {
		return nil, err
	}
	dynamicInterface = di
	return dynamicInterface, nil
}
