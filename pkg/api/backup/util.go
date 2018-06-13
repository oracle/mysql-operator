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

package backup

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/oracle/mysql-operator/pkg/apis/mysql/v1alpha1"
)

// GetBackupCondition extracts the provided condition from the given status and returns that.
// Returns nil and -1 if the condition is not present, and the index of the located condition.
func GetBackupCondition(status *v1alpha1.BackupStatus, conditionType v1alpha1.BackupConditionType) (int, *v1alpha1.BackupCondition) {
	if status == nil {
		return -1, nil
	}
	for i := range status.Conditions {
		if status.Conditions[i].Type == conditionType {
			return i, &status.Conditions[i]
		}
	}
	return -1, nil
}

// UpdateBackupCondition updates existing Backup condition or creates a new
// one. Sets LastTransitionTime to now if the status has changed.
// Returns true if Backup condition has changed or has been added.
func UpdateBackupCondition(status *v1alpha1.BackupStatus, condition *v1alpha1.BackupCondition) bool {
	condition.LastTransitionTime = metav1.Now()
	// Try to find this Backup condition.
	conditionIndex, oldCondition := GetBackupCondition(status, condition.Type)

	if oldCondition == nil {
		// We are adding new Backup condition.
		status.Conditions = append(status.Conditions, *condition)
		return true
	}
	// We are updating an existing condition, so we need to check if it has changed.
	if condition.Status == oldCondition.Status {
		condition.LastTransitionTime = oldCondition.LastTransitionTime
	}

	isEqual := condition.Status == oldCondition.Status &&
		condition.Reason == oldCondition.Reason &&
		condition.Message == oldCondition.Message &&
		condition.LastTransitionTime.Equal(&oldCondition.LastTransitionTime)

	status.Conditions[conditionIndex] = *condition
	// Return true if one of the fields have changed.
	return !isEqual
}

// IsBackupComplete returns true if a Backup has successfully completed
func IsBackupComplete(backup *v1alpha1.Backup) bool {
	return IsBackupCompleteConditionTrue(backup.Status)
}

// GetBackupCompleteCondition extracts the Backup complete condition from the given status and returns that.
// Returns nil if the condition is not present.
func GetBackupCompleteCondition(status v1alpha1.BackupStatus) *v1alpha1.BackupCondition {
	_, condition := GetBackupCondition(&status, v1alpha1.BackupComplete)
	return condition
}

// IsBackupCompleteConditionTrue returns true if a Backup is complete; false otherwise.
func IsBackupCompleteConditionTrue(status v1alpha1.BackupStatus) bool {
	condition := GetBackupCompleteCondition(status)
	return condition != nil && condition.Status == corev1.ConditionTrue
}
