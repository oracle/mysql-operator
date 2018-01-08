package v1

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

// Resource gets a MySQL Operator GroupResource for a specified resource.
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

// addKnownTypes adds the set of types defined in this package to the supplied
// scheme.
func addKnownTypes(s *runtime.Scheme) error {
	s.AddKnownTypes(SchemeGroupVersion,
		&MySQLCluster{},
		&MySQLClusterList{},
		&MySQLBackup{},
		&MySQLBackupList{},
		&MySQLRestore{},
		&MySQLRestoreList{},
		&MySQLBackupSchedule{},
		&MySQLBackupScheduleList{})
	metav1.AddToGroupVersion(s, SchemeGroupVersion)
	return nil
}
