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

package v1alpha1

// This package will auto register types with the Kubernetes API

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	// SchemeBuilder collects the scheme builder functions for the MySQL
	// Operator API.
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)

	// AddToScheme applies the SchemeBuilder functions to a specified scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

// GroupName is the group name for the MySQL Operator API.
const GroupName = "mysql.oracle.com"

// SchemeGroupVersion  is the GroupVersion for the MySQL Operator API.
var SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: "v1alpha1"}

const (
	// ClusterCRDResourceKind is the Kind of a Cluster.
	ClusterCRDResourceKind = "Cluster"
	// BackupCRDResourceKind is the Kind of a Backup.
	BackupCRDResourceKind = "Backup"
	// RestoreCRDResourceKind is the Kind of a Restore.
	RestoreCRDResourceKind = "Restore"
	// BackupScheduleCRDResourceKind is the Kind of a BackupSchedule.
	BackupScheduleCRDResourceKind = "BackupSchedule"
)

// Resource gets a MySQL Operator GroupResource for a specified resource.
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

// addKnownTypes adds the set of types defined in this package to the supplied
// scheme.
func addKnownTypes(s *runtime.Scheme) error {
	s.AddKnownTypes(SchemeGroupVersion,
		&Cluster{},
		&ClusterList{},
		&Backup{},
		&BackupList{},
		&Restore{},
		&RestoreList{},
		&BackupSchedule{},
		&BackupScheduleList{})
	metav1.AddToGroupVersion(s, SchemeGroupVersion)
	return nil
}
