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

// MinimumMySQLVersion is the minimum version of MySQL server supported by the
// MySQL Operator.
const MinimumMySQLVersion = "8.0.11"

// ClusterSpec defines the attributes a user can specify when creating a cluster
//+k8s:openapi-gen=true
type ClusterSpec struct {
	// Version defines the MySQL Docker image version.
	Version string `json:"version"`
	// Members defines the number of MySQL instances in a cluster
	Members int32 `json:"members,omitempty"`
	// BaseServerID defines the base number used to create unique server_id
	// for MySQL instances in the cluster. Valid range 1 to 4294967286.
	// If omitted in the manifest file (or set to 0) defaultBaseServerID
	// value will be used.
	BaseServerID uint32 `json:"baseServerID,omitempty"`
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
//+k8s:openapi-gen=true
type ClusterCondition struct {
	Type   ClusterConditionType   `json:"type"`
	Status corev1.ConditionStatus `json:"status"`
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime"`
	// +optional
	Reason string `json:"reason"`
	// +optional
	Message string `json:"message"`
}

// ClusterStatus defines the current status of a MySQL cluster
// propagating useful information back to the cluster admin
//+k8s:openapi-gen=true
type ClusterStatus struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	// +optional
	Conditions []ClusterCondition `json:"conditions"`
}

// +genclient
// +genclient:noStatus
// +resourceName=mysqlclusters
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Cluster represents a cluster spec and associated metadata
//+k8s:openapi-gen=true
type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   ClusterSpec   `json:"spec"`
	Status ClusterStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterList is a placeholder type for a list of MySQL clusters
//+k8s:openapi-gen=true
type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Cluster `json:"items"`
}

// Database represents a database to backup.
//+k8s:openapi-gen=true
type Database struct {
	Name string `json:"name"`
}

// MySQLDumpBackupExecutor executes backups using mysqldump.
//+k8s:openapi-gen=true
type MySQLDumpBackupExecutor struct {
	Databases []Database `json:"databases"`
}

// BackupExecutor represents the configuration of the tool performing the backup. This includes the tool
// to use, and, what database and tables should be backed up.
// The storage of the backup is configured in the relevant Storage configuration.
//+k8s:openapi-gen=true
type BackupExecutor struct {
	MySQLDump *MySQLDumpBackupExecutor `json:"mySQLDump"`
}

// S3StorageProvider represents an S3 compatible bucket for storing Backups.
//+k8s:openapi-gen=true
type S3StorageProvider struct {
	// Region in which the S3 compatible bucket is located.
	Region string `json:"region"`
	// Endpoint (hostname only or fully qualified URI) of S3 compatible
	// storage service.
	Endpoint string `json:"endpoint"`
	// Bucket in which to store the Backup.
	Bucket string `json:"bucket"`
	// ForcePathStyle when set to true forces the request to use path-style
	// addressing, i.e., `http://s3.amazonaws.com/BUCKET/KEY`. By default,
	// the S3 client will use virtual hosted bucket addressing when possible
	// (`http://BUCKET.s3.amazonaws.com/KEY`).
	ForcePathStyle bool `json:"forcePathStyle"`
	// CredentialsSecret is a reference to the Secret containing the
	// credentials authenticating with the S3 compatible storage service.
	CredentialsSecret *corev1.LocalObjectReference `json:"credentialsSecret"`
}

// StorageProvider defines the configuration for storing a Backup in a storage
// service.
//+k8s:openapi-gen=true
type StorageProvider struct {
	S3 *S3StorageProvider `json:"s3"`
}

// BackupSpec defines the specification for a MySQL backup. This includes what should be backed up,
// what tool should perform the backup, and, where the backup should be stored.
//+k8s:openapi-gen=true
type BackupSpec struct {
	// Executor is the configuration of the tool that will produce the backup, and a definition of
	// what databases and tables to backup.
	Executor BackupExecutor `json:"executor"`
	// StorageProvider configures where and how backups should be stored.
	StorageProvider StorageProvider `json:"storageProvider"`
	// Cluster is the Cluster to backup.
	Cluster *corev1.LocalObjectReference `json:"cluster"`
	// ScheduledMember is the Pod name of the Cluster member on which the
	// Backup will be executed.
	ScheduledMember string `json:"scheduledMember"`
}

// BackupConditionType represents a valid condition of a Backup.
type BackupConditionType string

const (
	// BackupScheduled means the Backup has been assigned to a Cluster
	// member for execution.
	BackupScheduled BackupConditionType = "Scheduled"
	// BackupRunning means the Backup is currently being executed by a
	// Cluster member's mysql-agent side-car.
	BackupRunning BackupConditionType = "Running"
	// BackupComplete means the Backup has successfully executed and the
	// resulting artifact has been stored in object storage.
	BackupComplete BackupConditionType = "Complete"
	// BackupFailed means the Backup has failed.
	BackupFailed BackupConditionType = "Failed"
)

// BackupCondition describes the observed state of a Backup at a certain point.
//+k8s:openapi-gen=true
type BackupCondition struct {
	Type   BackupConditionType    `json:"type"`
	Status corev1.ConditionStatus `json:"status"`
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime"`
	// +optional
	Reason string `json:"reason"`
	// +optional
	Message string `json:"message"`
}

// BackupOutcome describes the location of a Backup
//+k8s:openapi-gen=true
type BackupOutcome struct {
	// Location is the Object Storage network location of the Backup.
	Location string `json:"location"`
}

// BackupStatus captures the current status of a Backup.
//+k8s:openapi-gen=true
type BackupStatus struct {
	// Outcome holds the results of a successful backup.
	// +optional
	Outcome BackupOutcome `json:"outcome"`
	// TimeStarted is the time at which the backup was started.
	// +optional
	TimeStarted metav1.Time `json:"timeStarted"`
	// TimeCompleted is the time at which the backup completed.
	// +optional
	TimeCompleted metav1.Time `json:"timeCompleted"`
	// +optional
	Conditions []BackupCondition `json:"conditions"`
}

// +genclient
// +genclient:noStatus
// +resourceName=mysqlbackups
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Backup is a backup of a Cluster.
//+k8s:openapi-gen=true
type Backup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   BackupSpec   `json:"spec"`
	Status BackupStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BackupList is a list of Backups.
//+k8s:openapi-gen=true
type BackupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Backup `json:"items"`
}

// BackupScheduleSpec defines the specification for a MySQL backup schedule.
//+k8s:openapi-gen=true
type BackupScheduleSpec struct {
	// Schedule specifies the cron string used for backup scheduling.
	Schedule string `json:"schedule"`

	// BackupTemplate is the specification of the backup structure
	// to get scheduled.
	BackupTemplate BackupSpec `json:"backupTemplate"`
}

// ScheduleStatus captures the current state of a MySQL backup schedule.
//+k8s:openapi-gen=true
type ScheduleStatus struct {
	// LastBackup is the last time a Backup was run for this
	// backup schedule.
	// +optional
	LastBackup metav1.Time `json:"lastBackup"`
}

// +genclient
// +genclient:noStatus
// +resourceName=mysqlbackupschedules
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BackupSchedule is a backup schedule for a Cluster.
//+k8s:openapi-gen=true
type BackupSchedule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   BackupScheduleSpec `json:"spec"`
	Status ScheduleStatus     `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BackupScheduleList is a list of BackupSchedules.
//+k8s:openapi-gen=true
type BackupScheduleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []BackupSchedule `json:"items"`
}

// RestoreConditionType represents a valid condition of a Restore.
type RestoreConditionType string

const (
	// RestoreScheduled means the Restore has been assigned to a Cluster
	// member for execution.
	RestoreScheduled RestoreConditionType = "Scheduled"
	// RestoreRunning means the Restore is currently being executed by a
	// Cluster member's mysql-agent side-car.
	RestoreRunning RestoreConditionType = "Running"
	// RestoreComplete means the Restore has successfully executed and the
	// resulting artifact has been stored in object storage.
	RestoreComplete RestoreConditionType = "Complete"
	// RestoreFailed means the Restore has failed.
	RestoreFailed RestoreConditionType = "Failed"
)

// RestoreCondition describes the observed state of a Restore at a certain point.
//+k8s:openapi-gen=true
type RestoreCondition struct {
	Type   RestoreConditionType   `json:"type"`
	Status corev1.ConditionStatus `json:"status"`
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime"`
	// +optional
	Reason string `json:"reason"`
	// +optional
	Message string `json:"message"`
}

// RestoreSpec defines the specification for a restore of a MySQL backup.
//+k8s:openapi-gen=true
type RestoreSpec struct {
	// Cluster is a refeference to the Cluster to which the Restore
	// belongs.
	Cluster *corev1.LocalObjectReference `json:"cluster"`
	// Backup is a reference to the Backup object to be restored.
	Backup *corev1.LocalObjectReference `json:"backup"`
	// ScheduledMember is the Pod name of the Cluster member on which the
	// Restore will be executed.
	ScheduledMember string `json:"scheduledMember"`
}

// RestoreStatus captures the current status of a MySQL restore.
//+k8s:openapi-gen=true
type RestoreStatus struct {
	// TimeStarted is the time at which the restore was started.
	// +optional
	TimeStarted metav1.Time `json:"timeStarted"`
	// TimeCompleted is the time at which the restore completed.
	// +optional
	TimeCompleted metav1.Time `json:"timeCompleted"`
	// +optional
	Conditions []RestoreCondition `json:"conditions"`
}

// +genclient
// +genclient:noStatus
// +resourceName=mysqlrestores
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Restore is a MySQL Operator resource that represents the restoration of
// backup of a MySQL cluster.
//+k8s:openapi-gen=true
type Restore struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   RestoreSpec   `json:"spec"`
	Status RestoreStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RestoreList is a list of Restores.
//+k8s:openapi-gen=true
type RestoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Restore `json:"items"`
}
