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
	"strings"

	appsv1beta1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/oracle/mysql-operator/pkg/apis/mysql/v1alpha1"
	"github.com/oracle/mysql-operator/pkg/constants"
)

// SelectorForCluster creates a labels.Selector to match a given clusters
// associated resources.
func SelectorForCluster(c *v1alpha1.Cluster) labels.Selector {
	return labels.SelectorFromSet(labels.Set{constants.ClusterLabel: c.Name})
}

// SelectorForClusterOperatorVersion creates a labels.Selector to match a given clusters
// associated resources MySQLOperatorVersionLabel.
func SelectorForClusterOperatorVersion(operatorVersion string) labels.Selector {
	return labels.SelectorFromSet(labels.Set{constants.MySQLOperatorVersionLabel: operatorVersion})
}

func combineSelectors(first labels.Selector, rest ...labels.Selector) labels.Selector {
	res := first.DeepCopySelector()
	for _, s := range rest {
		reqs, _ := s.Requirements()
		res = res.Add(reqs...)
	}
	return res
}

func requiresMySQLAgentStatefulSetUpgrade(ss *appsv1beta1.StatefulSet, targetContainer string, operatorVersion string) bool {
	if !SelectorForClusterOperatorVersion(operatorVersion).Matches(labels.Set(ss.Labels)) {
		return true
	}
	for _, container := range ss.Spec.Template.Spec.Containers {
		if container.Name == targetContainer {
			parts := strings.Split(container.Image, ":")
			version := parts[len(parts)-1]
			return version != operatorVersion
		}
	}
	return false
}

func requiresMySQLAgentPodUpgrade(pod *corev1.Pod, targetContainer string, operatorVersion string) bool {
	if !SelectorForClusterOperatorVersion(operatorVersion).Matches(labels.Set(pod.Labels)) {
		return true
	}
	for _, container := range pod.Spec.Containers {
		if container.Name == targetContainer {
			parts := strings.Split(container.Image, ":")
			version := parts[len(parts)-1]
			return version != operatorVersion
		}
	}
	return false
}

// canUpgradeMySQLAgent checks that pod can actually be updated (e.g. there no backups currently taking place).
// TODO: Implement.
func canUpgradeMySQLAgent(pod *corev1.Pod) bool {
	return true
}
