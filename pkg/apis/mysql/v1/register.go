// Copyright 2018 Oracle and/or its affiliates. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1

// This package will auto register types with the Kubernetes API

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// GroupName is the group name for the MySQL Operator API.
const GroupName = "mysql.oracle.com"

var (
	// SchemeBuilder collects the scheme builder functions for the MySQL
	// Operator API.
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes, addDefaultingFuncs)

	// AddToScheme applies the SchemeBuilder functions to a specified scheme.
	AddToScheme        = SchemeBuilder.AddToScheme
	localSchemeBuilder = &SchemeBuilder
)

// SchemeGroupVersion  is the GroupVersion for the MySQL Operator API.
var SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: "v1"}

const (
	// MySQLClusterCRDResourceKind is the Kind of a MySQLCluster.
	MySQLClusterCRDResourceKind = "MySQLCluster"
	// MySQLBackupCRDResourceKind is the Kind of a MySQLBackup.
	MySQLBackupCRDResourceKind = "MySQLBackup"
	// MySQLRestoreCRDResourceKind is the Kind of a MySQLRestore.
	MySQLRestoreCRDResourceKind = "MySQLRestore"
	// MySQLBackupScheduleCRDResourceKind is the Kind of a MySQLBackupSchedule.
	MySQLBackupScheduleCRDResourceKind = "MySQLBackupSchedule"
)

// Kind takes an unqualified kind and returns back a Group qualified GroupKind
func Kind(kind string) schema.GroupKind {
	return SchemeGroupVersion.WithKind(kind).GroupKind()
}

// Resource gets a MySQL Operator GroupResource for a specified resource.
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

// addKnownTypes adds the set of types defined in this package to the supplied
// scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&MySQLCluster{},
		&MySQLClusterList{},
		&MySQLBackup{},
		&MySQLBackupList{},
		&MySQLRestore{},
		&MySQLRestoreList{},
		&MySQLBackupSchedule{},
		&MySQLBackupScheduleList{})
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	scheme.AddKnownTypes(SchemeGroupVersion, &metav1.Status{})
	return nil
}

func init() {
	// We only register manually written functions here. The registration of the
	// generated functions takes place in the generated files. The separation
	// makes the code compile even when the generated files are missing.
	localSchemeBuilder.Register(addKnownTypes)
}
