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
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestDefaultReplicas(t *testing.T) {
	cluster := &MySQLCluster{}
	cluster.EnsureDefaults()

	if cluster.Spec.Replicas != defaultReplicas {
		t.Errorf("Expected default replicas to be %d but got %d", defaultReplicas, cluster.Spec.Replicas)
	}
}

func TestDefaultVersion(t *testing.T) {
	cluster := &MySQLCluster{}
	cluster.EnsureDefaults()

	if cluster.Spec.Version != defaultVersion {
		t.Errorf("Expected default version to be %s but got %s", defaultVersion, cluster.Spec.Version)
	}
}

func TestRequiresConfigMount(t *testing.T) {
	cluster := &MySQLCluster{}
	cluster.EnsureDefaults()

	if cluster.RequiresConfigMount() {
		t.Errorf("Cluster without configRef should not require a config mount")
	}

	cluster = &MySQLCluster{
		Spec: MySQLClusterSpec{
			ConfigRef: &corev1.LocalObjectReference{
				Name: "customconfig",
			},
		},
	}

	if !cluster.RequiresConfigMount() {
		t.Errorf("Cluster with configRef should require a config mount")
	}
}
