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
	"github.com/oracle/mysql-operator/pkg/constants"
	"github.com/oracle/mysql-operator/pkg/version"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// RestoreCRDResourceSingular defines the singular custom resource name for
	// MySQLRestores.
	RestoreCRDResourceSingular = "mysqlrestore"

	// RestoreCRDResourcePlural defines the plural custom resource name for
	// MySQLRestores.
	RestoreCRDResourcePlural = "mysqlrestores"
)

// RestoreSpec defines the specification for a restore of a MySQL backup.
type RestoreSpec struct {
	// ClusterRef is a refeference to the MySQLCluster to which the MySQLRestore
	// belongs.
	ClusterRef *v1.LocalObjectReference `json:"clusterRef"`

	// BackupRef is a reference to the MySQLBackup object to be restored.
	BackupRef *v1.LocalObjectReference `json:"backupRef"`

	// AgentScheduled is the agent hostname to run the backup on
	AgentScheduled string `json:"agentscheduled"`
}

// RestorePhase represents the current life-cycle phase of a MySQLRestore.
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
	// Phase is the current life-cycle phase of the MySQLRestore.
	Phase RestorePhase `json:"phase"`

	// TimeStarted is the time at which the restore was started.
	TimeStarted metav1.Time `json:"timeStarted"`

	// TimeCompleted is the time at which the restore completed.
	TimeCompleted metav1.Time `json:"timeCompleted"`
}

// +genclient
// +genclient:noStatus

// MySQLRestore is a MySQL Operator resource that represents the restoration of
// backup of a MySQL cluster.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type MySQLRestore struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   RestoreSpec   `json:"spec"`
	Status RestoreStatus `json:"status"`
}

// MySQLRestoreList is a list of MySQLRestores.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type MySQLRestoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []MySQLRestore `json:"items"`
}

// EnsureDefaults can be invoked to ensure the default values are present.
func (r MySQLRestore) EnsureDefaults() *MySQLRestore {
	buildVersion := version.GetBuildVersion()
	if buildVersion != "" {
		if r.Labels == nil {
			r.Labels = make(map[string]string)
		}
		_, hasKey := r.Labels[constants.MySQLOperatorVersionLabel]
		if !hasKey {
			SetOperatorVersionLabel(r.Labels, buildVersion)
		}
	}
	return &r
}

// Validate checks if the resource spec is valid.
func (r MySQLRestore) Validate() error {
	return validateRestore(&r).ToAggregate()
}
