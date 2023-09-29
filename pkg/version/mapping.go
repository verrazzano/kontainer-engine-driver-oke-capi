// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package version

import (
	"context"
	"encoding/json"
	"errors"
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
	VerrazzanoTag     string `json:"-"`
}

type TagVersion struct {
	Tag     string
	Version string
}

func LoadDefaults(ctx context.Context, ki kubernetes.Interface) (*Defaults, error) {
	tagVersion, err := getVerrazzanoTagVersion(ctx, ki)
	if err != nil {
		return nil, err
	}
	if tagVersion == nil {
		return nil, errors.New("unknown Verrazzano defaults")
	}
	return &Defaults{
		VerrazzanoVersion: tagVersion.Version,
		VerrazzanoTag:     tagVersion.Tag,
	}, nil
}

func getVerrazzanoTagVersion(ctx context.Context, ki kubernetes.Interface) (*TagVersion, error) {
	cm, err := ki.CoreV1().ConfigMaps(verrazzanoInstallNamespace).Get(ctx, verrazzanoConfigMapName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if cm.Data == nil {
		return nil, errors.New("verrazzano-meta had no version data")
	}
	verrazzanoVersions := cm.Data["verrazzano-versions"]
	if len(verrazzanoVersions) < 1 {
		return nil, errors.New("verrazzano-meta had no verrazzano versions")
	}
	versionMapping := map[string]string{}
	if err := json.Unmarshal([]byte(verrazzanoVersions), &versionMapping); err != nil {
		return nil, err
	}
	if len(versionMapping) < 1 {
		return nil, errors.New("verrazzano version mapping was empty")
	}
	var versions []string
	for k := range versionMapping {
		versions = append(versions, k)
	}
	sort.Strings(versions)
	return &TagVersion{
		Tag:     versions[0],
		Version: versionMapping[versions[0]],
	}, nil
}
