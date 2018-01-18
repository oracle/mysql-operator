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

package labeler

import (
	labels "k8s.io/apimachinery/pkg/labels"
	selection "k8s.io/apimachinery/pkg/selection"

	constants "github.com/oracle/mysql-operator/pkg/constants"
)

// PrimarySelector returns a label selector that selects only primaries of a
// MySQLCluster's Pods.
func PrimarySelector(name string) labels.Selector {
	return labels.SelectorFromSet(labels.Set{
		constants.MySQLClusterLabel:     name,
		constants.LabelMySQLClusterRole: constants.MySQLClusterRolePrimary,
	})
}

// SecondarySelector returns a label selector that selects only secondaries of a
// MySQLCluster's Pods.
func SecondarySelector(name string) labels.Selector {
	return labels.SelectorFromSet(labels.Set{
		constants.MySQLClusterLabel:     name,
		constants.LabelMySQLClusterRole: constants.MySQLClusterRoleSecondary,
	})
}

// NonPrimarySelector returns a label selector that selects all Pods excluding
// primaries of a MySQLCluster.
func NonPrimarySelector(name string) labels.Selector {
	s := labels.SelectorFromSet(labels.Set{constants.MySQLClusterLabel: name})
	requirement, _ := labels.NewRequirement(constants.LabelMySQLClusterRole, selection.NotIn, []string{constants.MySQLClusterRolePrimary})
	return s.Add(*requirement)
}

// HasRoleSelector returns a label selector that selects all Pods for a
// MySQLCluster that have been labeled as having a role.
func HasRoleSelector(name string) labels.Selector {
	s := labels.SelectorFromSet(labels.Set{constants.MySQLClusterLabel: name})
	requirement, _ := labels.NewRequirement(constants.LabelMySQLClusterRole, selection.Exists, []string{})
	return s.Add(*requirement)
}
