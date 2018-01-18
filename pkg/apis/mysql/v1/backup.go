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

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/oracle/mysql-operator/pkg/constants"
	"github.com/oracle/mysql-operator/pkg/version"
)

const (
	// BackupCRDResourceSingular defines the singular custom resource name for MySQL backups.
	BackupCRDResourceSingular = "mysqlbackup"

	// BackupCRDResourcePlural defines the plural custom resource name for MySQL backups.
	BackupCRDResourcePlural = "mysqlbackups"
)

// BackupSpec defines the specification for a MySQL backup. This includes what should be backed up,
// what tool should perform the backup, and, where the backup should be stored.
type BackupSpec struct {
	// Executor is the configuration of the tool that will produce the backup, and a definition of
	// what databases and tables to backup.
	Executor *Executor `json:"executor"`

	// Storage is the configuration of where and how backups should be stored.
	Storage *Storage `json:"storage"`

	// ClusterRef is a reference to the MySQLCluster to which the MySQLBackup belongs.
	ClusterRef *corev1.LocalObjectReference `json:"clusterRef"`

	// AgentScheduled is the agent hostname to run the backup on
	AgentScheduled string `json:"agentscheduled"`
}

// Executor represents the configuration of the tool performing the backup. This includes the tool
// to use, and, what database and tables should be backed up.
// The storage of the backup is configured in the relevant Storage configuration.
type Executor struct {
	// The name of the tool performing the backup, e.g. mysqldump.
	Provider string `json:"provider"`
	// The databases to backup.
	Databases []string `json:"databases"`
}

// Storage defines the configuration for storing a MySQL backup to a storage service.
// The generation of the backup is configured in the Executor configuration.
type Storage struct {
	// Provider denotes the type of storage provider that will store and retrieve the backups,
	// e.g. s3, oci-s3-compat, aws-s3, gce-s3, etc.
	Provider string `json:"provider"`
	// SecretRef is a reference to the Kubernetes secret containing the configuration for uploading
	// the backup to authenticated storage.
	SecretRef *corev1.LocalObjectReference `json:"secretRef,omitempty"`
	// Config is generic string based key-value map that defines non-secret configuration values for
	// uploading the backup to storage w.r.t the configured storage provider.
	Config map[string]string `json:"config,omitempty"`
}

// BackupPhase represents the current life-cycle phase of a MySQLBackup.
type BackupPhase string

const (
	// BackupPhaseUnknown means that the backup hasn't yet been processed.
	BackupPhaseUnknown BackupPhase = ""

	// BackupPhaseNew means that the Backup hasn't yet been processed.
	BackupPhaseNew BackupPhase = "New"

	// BackupPhaseScheduled means that the Backup has been scheduled on an
	// appropriate replica.
	BackupPhaseScheduled BackupPhase = "Scheduled"

	// BackupPhaseStarted means the backup is in progress.
	BackupPhaseStarted BackupPhase = "Started"

	// BackupPhaseComplete means the backup has terminated successfully.
	BackupPhaseComplete BackupPhase = "Complete"

	// BackupPhaseFailed means the backup has terminated with an error.
	BackupPhaseFailed BackupPhase = "Failed"
)

// BackupOutcome describes the location of a MySQL Backup
type BackupOutcome struct {
	// Location is the Object Storage network location of the MySQLBackup.
	Location string `json:"location"`
}

// BackupStatus captures the current status of a MySQL backup.
type BackupStatus struct {
	// Phase is the current life-cycle phase of the MySQLBackup.
	Phase BackupPhase `json:"phase"`

	// Outcome holds the results of a successful backup.
	Outcome BackupOutcome `json:"outcome"`

	// TimeStarted is the time at which the backup was started.
	TimeStarted metav1.Time `json:"timeStarted"`

	// TimeCompleted is the time at which the backup completed.
	TimeCompleted metav1.Time `json:"timeCompleted"`
}

// +genclient
// +genclient:noStatus

// MySQLBackup is a MySQL Operator resource that represents a backup of a MySQL
// cluster.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type MySQLBackup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   BackupSpec   `json:"spec"`
	Status BackupStatus `json:"status"`
}

// MySQLBackupList is a list of MySQLBackups.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type MySQLBackupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []MySQLBackup `json:"items"`
}

// EnsureDefaults can be invoked to ensure the default values are present.
func (b MySQLBackup) EnsureDefaults() *MySQLBackup {
	buildVersion := version.GetBuildVersion()
	if buildVersion != "" {
		if b.Labels == nil {
			b.Labels = make(map[string]string)
		}
		_, hasKey := b.Labels[constants.MySQLOperatorVersionLabel]
		if !hasKey {
			SetOperatorVersionLabel(b.Labels, buildVersion)
		}
	}
	return &b
}

// Validate checks if the resource spec is valid.
func (b MySQLBackup) Validate() error {
	return validateBackup(&b).ToAggregate()
}
