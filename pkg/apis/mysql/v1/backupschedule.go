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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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

	// LastBackup is the last time a MySQLBackup was run for this
	// backup schedule.
	LastBackup metav1.Time `json:"lastBackup"`
}

// +genclient
// +genclient:noStatus

// MySQLBackupSchedule is a MySQL Operator resource that represents a backup
// schedule of a MySQL cluster.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type MySQLBackupSchedule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   BackupScheduleSpec `json:"spec"`
	Status ScheduleStatus     `json:"status,omitempty"`
}

// MySQLBackupScheduleList is a list of MySQLBackupSchedules.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type MySQLBackupScheduleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []MySQLBackupSchedule `json:"items"`
}

// EnsureDefaults can be invoked to ensure the default values are present.
func (b MySQLBackupSchedule) EnsureDefaults() *MySQLBackupSchedule {
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
func (b MySQLBackupSchedule) Validate() error {
	return validateBackupSchedule(&b).ToAggregate()
}
