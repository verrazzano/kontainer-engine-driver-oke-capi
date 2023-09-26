// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package version

import (
	"context"
	"encoding/json"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sort"
)

const (
	verrazzanoInstallNamespace = "verrazzano-install"
	verrazzanoConfigMapName    = "verrazzano-meta"
)

type Defaults struct {
	VerrazzanoVersion string `json:"-"`
}

func LoadDefaults(ctx context.Context, ki kubernetes.Interface) (*Defaults, error) {
	verrazzanoVersion, err := getVerrazzanoVersion(ctx, ki)
	if err != nil {
		return nil, err
	}
	return &Defaults{
		VerrazzanoVersion: verrazzanoVersion,
	}, nil
}

func getVerrazzanoVersion(ctx context.Context, ki kubernetes.Interface) (string, error) {
	cm, err := ki.CoreV1().ConfigMaps(verrazzanoInstallNamespace).Get(ctx, verrazzanoConfigMapName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return "", nil
		}
		return "", err
	}
	if cm.Data == nil {
		return "", nil
	}
	verrazzanoVersions := cm.Data["verrazzano-versions"]
	if len(verrazzanoVersions) < 1 {
		return "", nil
	}

	versionMapping := map[string]string{}
	if err := json.Unmarshal([]byte(verrazzanoVersions), &versionMapping); err != nil {
		return "", err
	}

	var versions []string
	for k := range versionMapping {
		versions = append(versions, k)
	}

	sort.Strings(versions)
	return versions[0], nil
}
