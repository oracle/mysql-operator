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

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// The default MySQL version to use if not specified explicitly by user
	defaultVersion      = "8.0.11"
	defaultReplicas     = 1
	defaultBaseServerID = 1000
	// Max safe value for BaseServerID calculated as max MySQL server_id value - max Replication Group size
	maxBaseServerID uint32 = 4294967295 - 9
)

const (
	// MaxInnoDBClusterMembers is the maximum number of members supported by InnoDB
	// group replication.
	MaxInnoDBClusterMembers = 9

	// ClusterNameMaxLen is the maximum supported length of a
	// Cluster name.
	// See: https://bugs.mysql.com/bug.php?id=90601
	ClusterNameMaxLen = 28
)

// TODO (owain) we need to remove this because it's not reasonable for us to maintain a list
// of all the potential MySQL versions that can be used and in reality, it shouldn't matter
// too much. The burden of this is not worth the benfit to a user
var validVersions = []string{
	defaultVersion,
}

// ClusterSpec defines the attributes a user can specify when creating a cluster
type ClusterSpec struct {
	// Version defines the MySQL Docker image version.
	Version string `json:"version"`

	// Replicas defines the number of running MySQL instances in a cluster
	Replicas int32 `json:"replicas,omitempty"`

	// BaseServerID defines the base number used to create uniq server_id for MySQL instances in a cluster.
	// The baseServerId value need to be in range from 1 to 4294967286
	// If ommited in the manifest file, or set to 0, defaultBaseServerID value will be used.
	BaseServerID uint32 `json:"baseServerId,omitempty"`

	// MultiMaster defines the mode of the MySQL cluster. If set to true,
	// all instances will be R/W. If false (the default), only a single instance
	// will be R/W and the rest will be R/O.
	MultiMaster bool `json:"multiMaster,omitempty"`

	// NodeSelector is a selector which must be true for the pod to fit on a node.
	// Selector which must match a node's labels for the pod to be scheduled on that node.
	// More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// If specified, affinity will define the pod's scheduling constraints
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// VolumeClaimTemplate allows a user to specify how volumes inside a MySQL cluster
	// +optional
	VolumeClaimTemplate *corev1.PersistentVolumeClaim `json:"volumeClaimTemplate,omitempty"`

	// BackupVolumeClaimTemplate allows a user to specify a volume to temporarily store the
	// data for a backup prior to it being shipped to object storage.
	// +optional
	BackupVolumeClaimTemplate *corev1.PersistentVolumeClaim `json:"backupVolumeClaimTemplate,omitempty"`

	// If defined, we use this secret for configuring the MYSQL_ROOT_PASSWORD
	// If it is not set we generate a secret dynamically
	// +optional
	RootPasswordSecret *corev1.LocalObjectReference `json:"rootPasswordSecret,omitempty"`

	// Config allows a user to specify a custom configuration file for MySQL.
	// +optional
	Config *corev1.LocalObjectReference `json:"config,omitempty"`

	// SSLSecret allows a user to specify custom CA certificate, server certificate
	// and server key for group replication SSL.
	// +optional
	SSLSecret *corev1.LocalObjectReference `json:"sslSecret,omitempty"`
}

// ClusterConditionType represents a valid condition of a Cluster.
type ClusterConditionType string

const (
	// ClusterReady means the Cluster is able to service requests.
	ClusterReady ClusterConditionType = "Ready"
)

// ClusterCondition describes the observed state of a Cluster at a certain point.
type ClusterCondition struct {
	Type   ClusterConditionType
	Status corev1.ConditionStatus
	// +optional
	LastTransitionTime metav1.Time
	// +optional
	Reason string
	// +optional
	Message string
}

// ClusterStatus defines the current status of a MySQL cluster
// propagating useful information back to the cluster admin
type ClusterStatus struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	// +optional
	Conditions []ClusterCondition
}

// +genclient
// +genclient:noStatus
// +resourceName=mysqlclusters
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Cluster represents a cluster spec and associated metadata
type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              ClusterSpec   `json:"spec"`
	Status            ClusterStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterList is a placeholder type for a list of MySQL clusters
type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Cluster `json:"items"`
}

// BackupSpec defines the specification for a MySQL backup. This includes what should be backed up,
// what tool should perform the backup, and, where the backup should be stored.
type BackupSpec struct {
	// Executor is the configuration of the tool that will produce the backup, and a definition of
	// what databases and tables to backup.
	Executor *BackupExecutor `json:"executor"`

	// StorageProvider is the configuration of where and how backups should be stored.
	StorageProvider *BackupStorageProvider `json:"storageProvider"`

	// Cluster is a reference to the Cluster to which the Backup belongs.
	Cluster *corev1.LocalObjectReference `json:"cluster"`

	// AgentScheduled is the agent hostname to run the backup on.
	// TODO(apryde): ScheduledAgent (*corev1.LocalObjectReference)?
	AgentScheduled string `json:"agentscheduled"`
}

// BackupExecutor represents the configuration of the tool performing the backup. This includes the tool
// to use, and, what database and tables should be backed up.
// The storage of the backup is configured in the relevant Storage configuration.
type BackupExecutor struct {
	// Name of the tool performing the backup, e.g. mysqldump.
	Name string `json:"name"`
	// Databases are the databases to backup.
	Databases []string `json:"databases"`
}

// BackupStorageProvider defines the configuration for storing a MySQL backup to a storage service.
// The generation of the backup is configured in the Executor configuration.
type BackupStorageProvider struct {
	// Name denotes the type of storage provider that will store and retrieve the backups.
	// Currently only supports "S3" denoting a S3 compatiable storage provider.
	Name string `json:"name"`
	// AuthSecret is a reference to the Kubernetes secret containing the configuration for uploading
	// the backup to authenticated storage.
	AuthSecret *corev1.LocalObjectReference `json:"authSecret,omitempty"`
	// Config is generic string based key-value map that defines non-secret configuration values for
	// uploading the backup to storage w.r.t the configured storage provider.
	Config map[string]string `json:"config,omitempty"`
}

// BackupPhase represents the current life-cycle phase of a Backup.
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
	// Location is the Object Storage network location of the Backup.
	Location string `json:"location"`
}

// BackupStatus captures the current status of a MySQL backup.
type BackupStatus struct {
	// Phase is the current life-cycle phase of the Backup.
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
// +resourceName=mysqlbackups
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Backup is a MySQL Operator resource that represents a backup of a MySQL
// cluster.
type Backup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   BackupSpec   `json:"spec"`
	Status BackupStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BackupList is a list of Backups.
type BackupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Backup `json:"items"`
}

// BackupScheduleSpec defines the specification for a MySQL backup schedule.
type BackupScheduleSpec struct {
	// Schedule specifies the cron string used for backup scheduling.
	Schedule string `json:"schedule"`

	// BackupTemplate is the specification of the backup structure
	// to get scheduled.
	BackupTemplate BackupSpec `json:"backupTemplate"`
}

// BackupSchedulePhase is a string representation of the lifecycle phase
// of a backup schedule.
type BackupSchedulePhase string

const (
	// BackupSchedulePhaseNew means the backup schedule has been created but not
	// yet processed by the backup schedule controller.
	BackupSchedulePhaseNew BackupSchedulePhase = "New"

	// BackupSchedulePhaseEnabled means the backup schedule has been validated and
	// will now be triggering backups according to the schedule spec.
	BackupSchedulePhaseEnabled BackupSchedulePhase = "Enabled"

	// BackupSchedulePhaseFailedValidation means the backup schedule has failed
	// the controller's validations and therefore will not trigger backups.
	BackupSchedulePhaseFailedValidation BackupSchedulePhase = "FailedValidation"
)

// ScheduleStatus captures the current state of a MySQL backup schedule.
type ScheduleStatus struct {
	// Phase is the current phase of the MySQL backup schedule.
	Phase BackupSchedulePhase `json:"phase"`

	// LastBackup is the last time a Backup was run for this
	// backup schedule.
	LastBackup metav1.Time `json:"lastBackup"`
}

// +genclient
// +genclient:noStatus
// +resourceName=mysqlbackupschedules
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BackupSchedule is a MySQL Operator resource that represents a backup
// schedule of a MySQL cluster.
type BackupSchedule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   BackupScheduleSpec `json:"spec"`
	Status ScheduleStatus     `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BackupScheduleList is a list of BackupSchedules.
type BackupScheduleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []BackupSchedule `json:"items"`
}

// RestoreSpec defines the specification for a restore of a MySQL backup.
type RestoreSpec struct {
	// Cluster is a refeference to the Cluster to which the Restore
	// belongs.
	Cluster *corev1.LocalObjectReference `json:"cluster"`

	// Backup is a reference to the Backup object to be restored.
	Backup *corev1.LocalObjectReference `json:"backup"`

	// AgentScheduled is the agent hostname to run the backup on
	AgentScheduled string `json:"agentscheduled"`
}

// RestorePhase represents the current life-cycle phase of a Restore.
type RestorePhase string

const (
	// RestorePhaseUnknown means that the restore hasn't yet been processed.
	RestorePhaseUnknown RestorePhase = ""

	// RestorePhaseNew means that the restore hasn't yet been processed.
	RestorePhaseNew RestorePhase = "New"

	// RestorePhaseScheduled means that the restore has been scheduled on an
	// appropriate replica.
	RestorePhaseScheduled RestorePhase = "Scheduled"

	// RestorePhaseStarted means the restore is in progress.
	RestorePhaseStarted RestorePhase = "Started"

	// RestorePhaseComplete means the restore has terminated successfully.
	RestorePhaseComplete RestorePhase = "Complete"

	// RestorePhaseFailed means the Restore has terminated with an error.
	RestorePhaseFailed RestorePhase = "Failed"
)

// RestoreStatus captures the current status of a MySQL restore.
type RestoreStatus struct {
	// Phase is the current life-cycle phase of the Restore.
	Phase RestorePhase `json:"phase"`

	// TimeStarted is the time at which the restore was started.
	TimeStarted metav1.Time `json:"timeStarted"`

	// TimeCompleted is the time at which the restore completed.
	TimeCompleted metav1.Time `json:"timeCompleted"`
}

// +genclient
// +genclient:noStatus
// +resourceName=mysqlrestores
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Restore is a MySQL Operator resource that represents the restoration of
// backup of a MySQL cluster.
type Restore struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   RestoreSpec   `json:"spec"`
	Status RestoreStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RestoreList is a list of Restores.
type RestoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Restore `json:"items"`
}
