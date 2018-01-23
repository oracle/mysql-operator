package v1

import (
	"k8s.io/apimachinery/pkg/runtime"
)

func addDefaultingFuncs(scheme *runtime.Scheme) error {
	return RegisterDefaults(scheme)
}

// SetDefaults_MySQLClusterSpec sets defaults for MySQLClusterSpec.
func SetDefaults_MySQLClusterSpec(spec *MySQLClusterSpec) {
	if spec.Replicas == 0 {
		spec.Replicas = 1
	}
}

// SetDefaults_MySQLClusterStatus sets defaults for MySQLClusterStatus.
func SetDefaults_MySQLClusterStatus(status *MySQLClusterStatus) {

}

// SetDefaults_BackupSpec sets defaults for BackupSpec.
func SetDefaults_BackupSpec(spec *BackupSpec) {

}

// SetDefaults_BackupStatus sets defaults for BackupStatus.
func SetDefaults_BackupStatus(status *BackupStatus) {

}

// SetDefaults_ScheduleSpec sets defaults for BackupScheduleSpec.
func SetDefaults_BackupScheduleSpec(spec *BackupScheduleSpec) {

}

// SetDefaults_ScheduleStatus sets defaults for ScheduleStatus.
func SetDefaults_ScheduleStatus(status *ScheduleStatus) {

}

// SetDefaults_RestoreSpec sets defaults for RestoreSpec.
func SetDefaults_RestoreSpec(spec *RestoreSpec) {

}

// SetDefaults_RestoreStatus sets defaults for RestoreStatus.
func SetDefaults_RestoreStatus(status *RestoreStatus) {

}
