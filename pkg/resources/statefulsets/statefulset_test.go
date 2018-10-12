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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/oracle/mysql-operator/pkg/apis/mysql/v1alpha1"
	operatoropts "github.com/oracle/mysql-operator/pkg/options/operator"
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

func TestClusterCustomSecurityContext(t *testing.T) {
	userID := int64(27)
	cluster := &v1alpha1.Cluster{
		Spec: v1alpha1.ClusterSpec{
			SecurityContext: &corev1.PodSecurityContext{
				RunAsUser: &userID,
				FSGroup:   &userID,
			},
		},
	}

	statefulSet := NewForCluster(cluster, mockOperatorConfig().Images, "mycluster")

	if assert.NotNil(t, statefulSet.Spec.Template.Spec.SecurityContext, "StatefulSet Spec is missing SecurityContext definition") {
		assert.EqualValues(t, userID, *statefulSet.Spec.Template.Spec.SecurityContext.RunAsUser, "SecurityContext Spec runAsUser does not have expected value")
		assert.Equal(t, userID, *statefulSet.Spec.Template.Spec.SecurityContext.FSGroup, "SecurityContext Spec fsGroup does not have expected value")
	}
}

func TestClusterWithTolerations(t *testing.T) {
	cluster := &v1alpha1.Cluster{
		Spec: v1alpha1.ClusterSpec{
			Tolerations: &[]corev1.Toleration{
				{
					Key:      "nodetype1",
					Operator: corev1.TolerationOpEqual,
					Value:    "true",
					Effect:   corev1.TaintEffectNoSchedule,
				},
				{
					Key:      "nodetype1",
					Operator: corev1.TolerationOpEqual,
					Value:    "true",
					Effect:   corev1.TaintEffectNoExecute,
				},
			},
		},
	}

	statefulSet := NewForCluster(cluster, mockOperatorConfig().Images, "mycluster")

	if assert.NotNil(t, statefulSet.Spec.Template.Spec.Tolerations, "StatefulSet Spec is missing Tolerations") {
		if assert.Len(t, statefulSet.Spec.Template.Spec.Tolerations, 2) {
			assert.Equal(t, "nodetype1", statefulSet.Spec.Template.Spec.Tolerations[0].Key, "ClusterSpec.Tolerations[0].Key does not have expected value")
			assert.Equal(t, corev1.TolerationOpEqual, statefulSet.Spec.Template.Spec.Tolerations[0].Operator, "ClusterSpec.Tolerations[0].Operator does not have expected value")
			assert.Equal(t, "true", statefulSet.Spec.Template.Spec.Tolerations[0].Value, "ClusterSpec.Tolerations[0].Value does not have expected value")
			assert.Equal(t, corev1.TaintEffectNoSchedule, statefulSet.Spec.Template.Spec.Tolerations[0].Effect, "ClusterSpec.Tolerations[0].Effect does not have expected value")

			assert.Equal(t, "nodetype1", statefulSet.Spec.Template.Spec.Tolerations[1].Key, "ClusterSpec.Tolerations[1].Key does not have expected value")
			assert.Equal(t, corev1.TolerationOpEqual, statefulSet.Spec.Template.Spec.Tolerations[1].Operator, "ClusterSpec.Tolerations[1].Operator does not have expected value")
			assert.Equal(t, "true", statefulSet.Spec.Template.Spec.Tolerations[1].Value, "ClusterSpec.Tolerations[1].Value does not have expected value")
			assert.Equal(t, corev1.TaintEffectNoExecute, statefulSet.Spec.Template.Spec.Tolerations[1].Effect, "ClusterSpec.Tolerations[1].Effect does not have expected value")
		}
	}
}

func TestClusterWithResourceRequirements(t *testing.T) {
	mysqlServerResourceRequirements := corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("2"),
			corev1.ResourceMemory: resource.MustParse("2Gi"),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("500m"),
			corev1.ResourceMemory: resource.MustParse("1Gi"),
		},
	}

	mysqlAgentResourceRequirements := corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("500m"),
			corev1.ResourceMemory: resource.MustParse("512Mi"),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("128Mi"),
		},
	}

	cluster := &v1alpha1.Cluster{
		Spec: v1alpha1.ClusterSpec{
			Resources: &v1alpha1.Resources{
				Server: &mysqlServerResourceRequirements,
				Agent:  &mysqlAgentResourceRequirements,
			},
		},
	}

	statefulSet := NewForCluster(cluster, mockOperatorConfig().Images, "mycluster")

	assert.Equal(t, mysqlServerResourceRequirements, statefulSet.Spec.Template.Spec.Containers[0].Resources, "MySQL-Server container resource requirements do not match expected.")
	assert.Equal(t, mysqlAgentResourceRequirements, statefulSet.Spec.Template.Spec.Containers[1].Resources, "MySQL-Agent container resource requirements do not match expected.")
}

func TestClusterWithOnlyMysqlServerResourceRequirements(t *testing.T) {
	mysqlServerResourceRequirements := corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("2"),
			corev1.ResourceMemory: resource.MustParse("2Gi"),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("500m"),
			corev1.ResourceMemory: resource.MustParse("1Gi"),
		},
	}

	cluster := &v1alpha1.Cluster{
		Spec: v1alpha1.ClusterSpec{
			Resources: &v1alpha1.Resources{
				Server: &mysqlServerResourceRequirements,
			},
		},
	}

	statefulSet := NewForCluster(cluster, mockOperatorConfig().Images, "mycluster")

	assert.Equal(t, mysqlServerResourceRequirements, statefulSet.Spec.Template.Spec.Containers[0].Resources, "MySQL-Server container resource requirements do not match expected.")
	assert.Nil(t, statefulSet.Spec.Template.Spec.Containers[1].Resources.Limits, "MySQL-Agent container has resource limits set which were not initially defined in the spec")
	assert.Nil(t, statefulSet.Spec.Template.Spec.Containers[1].Resources.Requests, "MySQL-Agent container has resource requests set which were not initially defined in the spec")

}

func TestClusterEnterpriseImage(t *testing.T) {
	cluster := &v1alpha1.Cluster{
		Spec: v1alpha1.ClusterSpec{
			Repository: "some/image/path",
			ImagePullSecrets: []corev1.LocalObjectReference{{
				Name: "someSecretName",
			}},
		},
	}
	cluster.EnsureDefaults()

	statefulSet := NewForCluster(cluster, mockOperatorConfig().Images, "mycluster")

	pullSecrets := statefulSet.Spec.Template.Spec.ImagePullSecrets
	ps := pullSecrets[len(pullSecrets)-1]
	si := statefulSet.Spec.Template.Spec.Containers[0].Image

	assert.Equal(t, "someSecretName", ps.Name)
	assert.Equal(t, "some/image/path:"+v1alpha1.DefaultVersion, si)
}

func TestClusterDefaultOverride(t *testing.T) {
	cluster := &v1alpha1.Cluster{}
	cluster.EnsureDefaults()
	cluster.Spec.Repository = "OverrideDefaultImage"

	operatorConf := mockOperatorConfig()
	operatorConf.Images.DefaultMySQLServerImage = "newDefaultImage"
	statefulSet := NewForCluster(cluster, operatorConf.Images, "mycluster")

	si := statefulSet.Spec.Template.Spec.Containers[0].Image

	assert.Equal(t, "OverrideDefaultImage:"+v1alpha1.DefaultVersion, si)
}
