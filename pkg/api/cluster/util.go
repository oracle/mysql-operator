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

package cluster

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/oracle/mysql-operator/pkg/apis/mysql/v1alpha1"
)

// GetClusterCondition extracts the provided condition from the given status and returns that.
// Returns nil and -1 if the condition is not present, and the index of the located condition.
func GetClusterCondition(status *v1alpha1.ClusterStatus, conditionType v1alpha1.ClusterConditionType) (int, *v1alpha1.ClusterCondition) {
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

// UpdateClusterCondition updates existing Cluster condition or creates a new
// one. Sets LastTransitionTime to now if the status has changed.
// Returns true if Cluster condition has changed or has been added.
func UpdateClusterCondition(status *v1alpha1.ClusterStatus, condition *v1alpha1.ClusterCondition) bool {
	condition.LastTransitionTime = metav1.Now()
	// Try to find this Cluster condition.
	conditionIndex, oldCondition := GetClusterCondition(status, condition.Type)

	if oldCondition == nil {
		// We are adding new Cluster condition.
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

// IsClusterReady returns true if a Cluster is ready; false otherwise.
func IsClusterReady(cluster *v1alpha1.Cluster) bool {
	return IsClusterReadyConditionTrue(cluster.Status)
}

// GetClusterReadyCondition extracts the Cluster ready condition from the given status and returns that.
// Returns nil if the condition is not present.
func GetClusterReadyCondition(status v1alpha1.ClusterStatus) *v1alpha1.ClusterCondition {
	_, condition := GetClusterCondition(&status, v1alpha1.ClusterReady)
	return condition
}

// IsClusterReadyConditionTrue returns true if a Cluster is ready; false otherwise.
func IsClusterReadyConditionTrue(status v1alpha1.ClusterStatus) bool {
	condition := GetClusterReadyCondition(status)
	return condition != nil && condition.Status == corev1.ConditionTrue
}
