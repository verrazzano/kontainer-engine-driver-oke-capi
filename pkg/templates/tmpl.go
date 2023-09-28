// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package templates

import _ "embed"

//go:embed cluster.goyaml
var Cluster string

//go:embed ocimanagedcluster.goyaml
var OCIManagedCluster string

//go:embed ociclusteridentity.goyaml
var ClusterIdentity string

//go:embed ocimanagedcontrolplane.goyaml
var OCIManagedControlPlane string

//go:embed machinepool.goyaml
var MachinePool string

//go:embed ocimanagedmachinepool.goyaml
var OCIManagedMachinePool string

//go:embed verrazzanofleet.goyaml
var VerrazzanoFleet string

//go:embed imagepullsecret.goyaml
var ImagePullSecret string
