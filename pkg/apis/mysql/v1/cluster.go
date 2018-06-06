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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	// The default MySQL version to use if not specified explicitly by user
	defaultVersion  = "5.7.20-1.1.2"
	defaultReplicas = 1
)

// ClusterCRDResourcePlural defines the custom resource name for mysqlclusters
const ClusterCRDResourcePlural = "mysqlclusters"

// MySQLClusterSpec defines the attributes a user can specify when creating a cluster
type MySQLClusterSpec struct {
	// Version defines the MySQL Docker image version.
	Version string `json:"version"`

	// Replicas defines the number of running MySQL instances in a cluster
	Replicas int32 `json:"replicas,omitempty"`

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
	SecretRef *corev1.LocalObjectReference `json:"secretRef,omitempty"`

	// ConfigRef allows a user to specify a custom configuration file for MySQL.
	// +optional
	ConfigRef *corev1.LocalObjectReference `json:"configRef,omitempty"`
}

// MySQLClusterPhase describes the state of the cluster.
type MySQLClusterPhase string

const (
	// MySQLClusterPending means the cluster has been accepted by the system,
	// but one or more of the services or statefulsets has not been started.
	// This includes time before being bound to a node, as well as time spent
	// pulling images onto the host.
	MySQLClusterPending MySQLClusterPhase = "Pending"

	// MySQLClusterRunning means the cluster has been created, all of it's
	// required components are present, and there is at least one endpoint that
	// mysql client can connect to.
	MySQLClusterRunning MySQLClusterPhase = "Running"

	// MySQLClusterSucceeded means that all containers in the pod have
	// voluntarily terminated with a container exit code of 0, and the system
	// is not going to restart any of these containers.
	MySQLClusterSucceeded MySQLClusterPhase = "Succeeded"

	// MySQLClusterFailed means that all containers in the pod have terminated,
	// and at least one container has terminated in a failure (exited with a
	// non-zero exit code or was stopped by the system).
	MySQLClusterFailed MySQLClusterPhase = "Failed"

	// MySQLClusterUnknown means that for some reason the state of the cluster
	// could not be obtained, typically due to an error in communicating with
	// the host of the pod.
	MySQLClusterUnknown MySQLClusterPhase = ""
)

// MySQLClusterValidPhases denote the life-cycle states a cluster can be in.
var MySQLClusterValidPhases = []MySQLClusterPhase{
	MySQLClusterPending,
	MySQLClusterRunning,
	MySQLClusterSucceeded,
	MySQLClusterFailed,
	MySQLClusterUnknown}

// MySQLClusterStatus defines the current status of a MySQL cluster
// propagating useful information back to the cluster admin
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type MySQLClusterStatus struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Phase             MySQLClusterPhase `json:"phase"`
	Errors            []string          `json:"errors"`
}

// +genclient
// +genclient:noStatus

// MySQLCluster represents a cluster spec and associated metadata
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type MySQLCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              MySQLClusterSpec   `json:"spec"`
	Status            MySQLClusterStatus `json:"status"`
}

// MySQLClusterList is a placeholder type for a list of MySQL clusters
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type MySQLClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []MySQLCluster `json:"items"`
}

// Validate returns an error if a cluster is invalid
func (c *MySQLCluster) Validate() error {
	return validateCluster(c).ToAggregate()
}

// EnsureDefaults will ensure that if a user omits and fields in the
// spec that are required, we set some sensible defaults.
// For example a user can choose to omit the version
// and number of replics
func (c *MySQLCluster) EnsureDefaults() *MySQLCluster {
	if c.Spec.Replicas == 0 {
		c.Spec.Replicas = defaultReplicas
	}

	if c.Spec.Version == "" {
		c.Spec.Version = defaultVersion
	}

	return c
}

// RequiresConfigMount will return true if a user has specified a config map
// for configuring the cluster else false
func (c *MySQLCluster) RequiresConfigMount() bool {
	return c.Spec.ConfigRef != nil
}

// RequiresSecret returns true if a secret should be generated
// for a MySQL cluster else false
func (c *MySQLCluster) RequiresSecret() bool {
	return c.Spec.SecretRef == nil
}

// GetObjectKind is required for codegen
func (c *MySQLCluster) GetObjectKind() schema.ObjectKind {
	return &c.TypeMeta
}

// GetObjectKind is required for codegen
func (c *MySQLClusterStatus) GetObjectKind() schema.ObjectKind {
	return &c.TypeMeta
}

// GetObjectKind is required for codegen
func (c *MySQLClusterList) GetObjectKind() schema.ObjectKind {
	return &c.TypeMeta
}
