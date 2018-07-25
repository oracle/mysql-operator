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

package statefulsets

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	operatoropts "github.com/oracle/mysql-operator/pkg/options/operator"
	"github.com/oracle/mysql-operator/pkg/apis/mysql/v1alpha1"
)

func mockOperatorConfig() operatoropts.MySQLOperatorOpts {
	opts := operatoropts.MySQLOperatorOpts{}
	opts.EnsureDefaults()
	return opts
}

func TestMySQLRootPasswordNoSecretRef(t *testing.T) {
	cluster := &v1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
		Spec:       v1alpha1.ClusterSpec{},
	}

	actual := mysqlRootPassword(cluster).ValueFrom.SecretKeyRef.Name

	if actual != "cluster-root-password" {
		t.Errorf("Expected cluster-root-password but got %s", actual)
	}
}

func TestMySQLRootPasswordWithSecretRef(t *testing.T) {
	cluster := &v1alpha1.Cluster{
		Spec: v1alpha1.ClusterSpec{
			RootPasswordSecret: &corev1.LocalObjectReference{Name: "secret"},
		},
	}

	actual := mysqlRootPassword(cluster).ValueFrom.SecretKeyRef.Name

	if actual != "secret" {
		t.Errorf("Expected secret but got %s", actual)
	}
}

func TestClusterWithoutPVCHasBackupContainerAndVolumes(t *testing.T) {
	cluster := &v1alpha1.Cluster{
		Spec: v1alpha1.ClusterSpec{
			RootPasswordSecret: &corev1.LocalObjectReference{Name: "secret"},
		},
	}

	statefulSet := NewForCluster(cluster, mockOperatorConfig().Images, "mycluster")
	containers := statefulSet.Spec.Template.Spec.Containers
	volumes := statefulSet.Spec.Template.Spec.Volumes
	if len(volumes) != 2 {
		t.Errorf("Expected two volumes but found %d", len(volumes))
	}

	if len(containers) != 2 {
		t.Errorf("Expected two containers but found %d", len(containers))
	}
}

func TestClusterWithPVCHasBackupContainerAndVolumes(t *testing.T) {
	cluster := &v1alpha1.Cluster{
		Spec: v1alpha1.ClusterSpec{
			RootPasswordSecret:        &corev1.LocalObjectReference{Name: "secret"},
			VolumeClaimTemplate:       &corev1.PersistentVolumeClaim{},
			BackupVolumeClaimTemplate: &corev1.PersistentVolumeClaim{},
		},
	}

	statefulSet := NewForCluster(cluster, mockOperatorConfig().Images, "mycluster")
	containers := statefulSet.Spec.Template.Spec.Containers
	volumes := statefulSet.Spec.Template.Spec.Volumes
	if len(volumes) != 0 {
		t.Errorf("Expected zero volumes but found %d", len(volumes))
	}

	if len(containers) != 2 {
		t.Errorf("Expected two containers but found %d", len(containers))
	}
}

func TestClusterHasNodeSelector(t *testing.T) {
	nvmeSelector := map[string]string{"disk": "nvme"}
	cluster := &v1alpha1.Cluster{
		Spec: v1alpha1.ClusterSpec{
			NodeSelector: nvmeSelector,
		},
	}

	statefulSet := NewForCluster(cluster, mockOperatorConfig().Images, "mycluster")

	if !reflect.DeepEqual(statefulSet.Spec.Template.Spec.NodeSelector, nvmeSelector) {
		t.Errorf("Expected cluster with NVMe node selector")
	}
}

func TestClusterCustomConfig(t *testing.T) {
	cluster := &v1alpha1.Cluster{
		Spec: v1alpha1.ClusterSpec{
			Config: &corev1.LocalObjectReference{
				Name: "mycnf",
			},
		},
	}

	statefulSet := NewForCluster(cluster, mockOperatorConfig().Images, "mycluster")
	containers := statefulSet.Spec.Template.Spec.Containers

	var hasExpectedVolumeMount = false
	for _, container := range containers {
		for _, mount := range container.VolumeMounts {
			if mount.MountPath == "/etc/my.cnf" {
				hasExpectedVolumeMount = true
				break
			}
		}
	}

	if !hasExpectedVolumeMount {
		t.Errorf("Cluster is missing expected volume mount for custom config map")
	}
}

func TestClusterCustomSSLSetup(t *testing.T) {
	cluster := &v1alpha1.Cluster{
		Spec: v1alpha1.ClusterSpec{
			SSLSecret: &corev1.LocalObjectReference{
				Name: "my-ssl",
			},
		},
	}

	statefulSet := NewForCluster(cluster, mockOperatorConfig().Images, "mycluster")
	containers := statefulSet.Spec.Template.Spec.Containers

	var hasExpectedVolumeMount = false
	for _, container := range containers {
		for _, mount := range container.VolumeMounts {
			if mount.MountPath == "/etc/ssl/mysql" {
				hasExpectedVolumeMount = true
				break
			}
		}
	}

	assert.True(t, hasExpectedVolumeMount, "Cluster is missing expected volume mount for custom SSL certs")
}
