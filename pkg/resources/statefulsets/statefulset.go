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
	"fmt"
	"os"
	"strconv"
	"strings"

	apps "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/oracle/mysql-operator/pkg/apis/mysql/v1alpha1"
	"github.com/oracle/mysql-operator/pkg/constants"
	agentopts "github.com/oracle/mysql-operator/pkg/options/agent"
	operatoropts "github.com/oracle/mysql-operator/pkg/options/operator"
	"github.com/oracle/mysql-operator/pkg/resources/secrets"
	"github.com/oracle/mysql-operator/pkg/version"

	"github.com/coreos/go-semver/semver"
)

const (
	// MySQLServerName is the static name of all 'mysql(-server)' containers.
	MySQLServerName = "mysql"
	// MySQLAgentName is the static name of all 'mysql-agent' containers.
	MySQLAgentName = "mysql-agent"
	// MySQLAgentBasePath defines the volume mount path for the MySQL agent
	MySQLAgentBasePath = "/var/lib/mysql-agent"

	mySQLBackupVolumeName = "mysqlbackupvolume"
	mySQLVolumeName       = "mysqlvolume"
	mySQLSSLVolumeName    = "mysqlsslvolume"

	replicationGroupPort = 13306

	minMysqlVersionWithGroupExitStateArgs = "8.0.12"
)

func volumeMounts(cluster *v1alpha1.Cluster) []v1.VolumeMount {
	var mounts []v1.VolumeMount

	name := mySQLVolumeName
	if cluster.Spec.VolumeClaimTemplate != nil {
		name = cluster.Spec.VolumeClaimTemplate.Name
	}

	mounts = append(mounts, v1.VolumeMount{
		Name:      name,
		MountPath: "/var/lib/mysql",
		SubPath:   "mysql",
	})

	backupName := mySQLBackupVolumeName
	if cluster.Spec.BackupVolumeClaimTemplate != nil {
		backupName = cluster.Spec.BackupVolumeClaimTemplate.Name
	}
	mounts = append(mounts, v1.VolumeMount{
		Name:      backupName,
		MountPath: MySQLAgentBasePath,
		SubPath:   "mysql",
	})

	// A user may explicitly define a my.cnf configuration file for
	// their MySQL cluster.
	if cluster.RequiresConfigMount() {
		mounts = append(mounts, v1.VolumeMount{
			Name:      cluster.Name,
			MountPath: "/etc/my.cnf",
			SubPath:   "my.cnf",
		})
	}

	if cluster.RequiresCustomSSLSetup() {
		mounts = append(mounts, v1.VolumeMount{
			Name:      mySQLSSLVolumeName,
			MountPath: "/etc/ssl/mysql",
		})
	}

	return mounts
}

func clusterNameEnvVar(cluster *v1alpha1.Cluster) v1.EnvVar {
	return v1.EnvVar{Name: "MYSQL_CLUSTER_NAME", Value: cluster.Name}
}

func namespaceEnvVar() v1.EnvVar {
	return v1.EnvVar{
		Name: "POD_NAMESPACE",
		ValueFrom: &v1.EnvVarSource{
			FieldRef: &v1.ObjectFieldSelector{
				FieldPath: "metadata.namespace",
			},
		},
	}
}

func replicationGroupSeedsEnvVar(replicationGroupSeeds string) v1.EnvVar {
	return v1.EnvVar{
		Name:  "REPLICATION_GROUP_SEEDS",
		Value: replicationGroupSeeds,
	}
}

func multiMasterEnvVar(enabled bool) v1.EnvVar {
	return v1.EnvVar{
		Name:  "MYSQL_CLUSTER_MULTI_MASTER",
		Value: strconv.FormatBool(enabled),
	}
}

// Returns the MySQL_ROOT_PASSWORD environment variable
// If a user specifies a secret in the spec we use that
// else we create a secret with a random password
func mysqlRootPassword(cluster *v1alpha1.Cluster) v1.EnvVar {
	var secretName string
	if cluster.RequiresSecret() {
		secretName = secrets.GetRootPasswordSecretName(cluster)
	} else {
		secretName = cluster.Spec.RootPasswordSecret.Name
	}

	return v1.EnvVar{
		Name: "MYSQL_ROOT_PASSWORD",
		ValueFrom: &v1.EnvVarSource{
			SecretKeyRef: &v1.SecretKeySelector{
				LocalObjectReference: v1.LocalObjectReference{
					Name: secretName,
				},
				Key: "password",
			},
		},
	}
}

func getReplicationGroupSeeds(name string, members int) string {
	seeds := []string{}
	for i := 0; i < members; i++ {
		seeds = append(seeds, fmt.Sprintf("%[1]s-%[2]d.%[1]s:%[3]d", name, i, replicationGroupPort))
	}
	return strings.Join(seeds, ",")
}

func checkSupportGroupExitStateArgs(deployingVersion string) (supportedVer bool) {
	defer func() {
		if r := recover(); r != nil {

		}
	}()

	supportedVer = false

	ver := semver.New(deployingVersion)
	minVer := semver.New(minMysqlVersionWithGroupExitStateArgs)

	if ver.LessThan(*minVer) {
		return
	}

	supportedVer = true
	return
}

// Builds the MySQL operator container for a cluster.
// The 'mysqlImage' parameter is the image name of the mysql server to use with
// no version information.. e.g. 'mysql/mysql-server'
func mysqlServerContainer(cluster *v1alpha1.Cluster, mysqlServerImage string, rootPassword v1.EnvVar, members int, baseServerID uint32) v1.Container {
	args := []string{
		"--server_id=$(expr $base + $index)",
		"--datadir=/var/lib/mysql",
		"--user=mysql",
		"--gtid_mode=ON",
		"--log-bin",
		"--binlog_checksum=NONE",
		"--enforce_gtid_consistency=ON",
		"--log-slave-updates=ON",
		"--binlog-format=ROW",
		"--master-info-repository=TABLE",
		"--relay-log-info-repository=TABLE",
		"--transaction-write-set-extraction=XXHASH64",
		fmt.Sprintf("--relay-log=%s-${index}-relay-bin", cluster.Name),
		fmt.Sprintf("--report-host=\"%[1]s-${index}.%[1]s\"", cluster.Name),
		"--log-error-verbosity=3",
	}

	if cluster.RequiresCustomSSLSetup() {
		args = append(args,
			"--ssl-ca=/etc/ssl/mysql/ca.crt",
			"--ssl-cert=/etc/ssl/mysql/tls.crt",
			"--ssl-key=/etc/ssl/mysql/tls.key")
	}

	if checkSupportGroupExitStateArgs(cluster.Spec.Version) {
		args = append(args, "--loose-group-replication-exit-state-action=READ_ONLY")
	}

	entryPointArgs := strings.Join(args, " ")

	cmd := fmt.Sprintf(`
         # Set baseServerID
         base=%d

         # Finds the replica index from the hostname, and uses this to define
         # a unique server id for this instance.
         index=$(cat /etc/hostname | grep -o '[^-]*$')
         /entrypoint.sh %s`, baseServerID, entryPointArgs)

	var resourceLimits corev1.ResourceRequirements
	if cluster.Spec.Resources != nil && cluster.Spec.Resources.Server != nil {
		resourceLimits = *cluster.Spec.Resources.Server
	}

	return v1.Container{
		Name: MySQLServerName,
		// TODO(apryde): Add BaseImage to cluster CRD.
		Image: fmt.Sprintf("%s:%s", mysqlServerImage, cluster.Spec.Version),
		Ports: []v1.ContainerPort{
			{
				ContainerPort: 3306,
			},
		},
		VolumeMounts: volumeMounts(cluster),
		Command:      []string{"/bin/bash", "-ecx", cmd},
		Env: []v1.EnvVar{
			rootPassword,
			{
				Name:  "MYSQL_ROOT_HOST",
				Value: "%",
			},
			{
				Name:  "MYSQL_LOG_CONSOLE",
				Value: "true",
			},
		},
		Resources: resourceLimits,
	}
}

func mysqlAgentContainer(cluster *v1alpha1.Cluster, mysqlAgentImage string, rootPassword v1.EnvVar, members int) v1.Container {
	agentVersion := version.GetBuildVersion()
	if v := os.Getenv("MYSQL_AGENT_VERSION"); v != "" {
		agentVersion = v
	}

	replicationGroupSeeds := getReplicationGroupSeeds(cluster.Name, members)

	var resourceLimits corev1.ResourceRequirements
	if cluster.Spec.Resources != nil && cluster.Spec.Resources.Agent != nil {
		resourceLimits = *cluster.Spec.Resources.Agent
	}

	return v1.Container{
		Name:         MySQLAgentName,
		Image:        fmt.Sprintf("%s:%s", mysqlAgentImage, agentVersion),
		Args:         []string{"--v=4"},
		VolumeMounts: volumeMounts(cluster),
		Env: []v1.EnvVar{
			clusterNameEnvVar(cluster),
			namespaceEnvVar(),
			replicationGroupSeedsEnvVar(replicationGroupSeeds),
			multiMasterEnvVar(cluster.Spec.MultiMaster),
			rootPassword,
			{
				Name: "MY_POD_IP",
				ValueFrom: &v1.EnvVarSource{
					FieldRef: &v1.ObjectFieldSelector{
						FieldPath: "status.podIP",
					},
				},
			},
		},
		LivenessProbe: &v1.Probe{
			Handler: v1.Handler{
				HTTPGet: &v1.HTTPGetAction{
					Path: "/live",
					Port: intstr.FromInt(int(agentopts.DefaultMySQLAgentHeathcheckPort)),
				},
			},
		},
		ReadinessProbe: &v1.Probe{
			Handler: v1.Handler{
				HTTPGet: &v1.HTTPGetAction{
					Path: "/ready",
					Port: intstr.FromInt(int(agentopts.DefaultMySQLAgentHeathcheckPort)),
				},
			},
		},
		Resources: resourceLimits,
	}
}

// NewForCluster creates a new StatefulSet for the given Cluster.
func NewForCluster(cluster *v1alpha1.Cluster, images operatoropts.Images, serviceName string) *apps.StatefulSet {
	rootPassword := mysqlRootPassword(cluster)
	members := int(cluster.Spec.Members)
	baseServerID := cluster.Spec.BaseServerID

	// If a PV isn't specified just use a EmptyDir volume
	var podVolumes = []v1.Volume{}
	if cluster.Spec.VolumeClaimTemplate == nil {
		podVolumes = append(podVolumes, v1.Volume{Name: mySQLVolumeName,
			VolumeSource: v1.VolumeSource{EmptyDir: &v1.EmptyDirVolumeSource{Medium: ""}}})
	}

	// If a Backup PV isn't specified just use a EmptyDir volume
	if cluster.Spec.BackupVolumeClaimTemplate == nil {
		podVolumes = append(podVolumes, v1.Volume{Name: mySQLBackupVolumeName,
			VolumeSource: v1.VolumeSource{EmptyDir: &v1.EmptyDirVolumeSource{Medium: ""}}})
	}

	if cluster.RequiresConfigMount() {
		podVolumes = append(podVolumes, v1.Volume{
			Name: cluster.Name,
			VolumeSource: v1.VolumeSource{
				ConfigMap: &v1.ConfigMapVolumeSource{
					LocalObjectReference: v1.LocalObjectReference{
						Name: cluster.Spec.Config.Name,
					},
				},
			},
		})
	}

	if cluster.RequiresCustomSSLSetup() {
		podVolumes = append(podVolumes, v1.Volume{
			Name: mySQLSSLVolumeName,
			VolumeSource: v1.VolumeSource{
				Projected: &v1.ProjectedVolumeSource{
					Sources: []v1.VolumeProjection{
						{
							Secret: &v1.SecretProjection{
								LocalObjectReference: v1.LocalObjectReference{
									Name: cluster.Spec.SSLSecret.Name,
								},
								Items: []v1.KeyToPath{
									{
										Key:  "ca.crt",
										Path: "ca.crt",
									},
									{
										Key:  "tls.crt",
										Path: "tls.crt",
									},
									{
										Key:  "tls.key",
										Path: "tls.key",
									},
								},
							},
						},
					},
				},
			},
		})
	}

	containers := []v1.Container{
		mysqlServerContainer(cluster, cluster.Spec.Repository, rootPassword, members, baseServerID),
		mysqlAgentContainer(cluster, images.MySQLAgentImage, rootPassword, members)}

	podLabels := map[string]string{
		constants.ClusterLabel: cluster.Name,
	}
	if cluster.Spec.MultiMaster {
		podLabels[constants.LabelClusterRole] = constants.ClusterRolePrimary
	}

	ss := &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: cluster.Namespace,
			Name:      cluster.Name,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(cluster, schema.GroupVersionKind{
					Group:   v1alpha1.SchemeGroupVersion.Group,
					Version: v1alpha1.SchemeGroupVersion.Version,
					Kind:    v1alpha1.ClusterCRDResourceKind,
				}),
			},
			Labels: map[string]string{
				constants.ClusterLabel:              cluster.Name,
				constants.MySQLOperatorVersionLabel: version.GetBuildVersion(),
			},
		},
		Spec: apps.StatefulSetSpec{
			Replicas: &cluster.Spec.Members,
			Selector: &metav1.LabelSelector{
				MatchLabels: podLabels,
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: podLabels,
					Annotations: map[string]string{
						"prometheus.io/scrape": "true",
						"prometheus.io/port":   "8080",
					},
				},
				Spec: v1.PodSpec{
					// FIXME: LIMITED TO DEFAULT NAMESPACE. Need to dynamically
					// create service accounts and (cluster role bindings?)
					// for each namespace.
					ServiceAccountName: "mysql-agent",
					NodeSelector:       cluster.Spec.NodeSelector,
					Affinity:           cluster.Spec.Affinity,
					Containers:         containers,
					Volumes:            podVolumes,
				},
			},
			UpdateStrategy: apps.StatefulSetUpdateStrategy{
				Type: apps.RollingUpdateStatefulSetStrategyType,
			},
			ServiceName: serviceName,
		},
	}

	if cluster.Spec.ImagePullSecrets != nil {
		ss.Spec.Template.Spec.ImagePullSecrets = append(ss.Spec.Template.Spec.ImagePullSecrets, cluster.Spec.ImagePullSecrets...)
	}
	if cluster.Spec.VolumeClaimTemplate != nil {
		ss.Spec.VolumeClaimTemplates = append(ss.Spec.VolumeClaimTemplates, *cluster.Spec.VolumeClaimTemplate)
	}
	if cluster.Spec.BackupVolumeClaimTemplate != nil {
		ss.Spec.VolumeClaimTemplates = append(ss.Spec.VolumeClaimTemplates, *cluster.Spec.BackupVolumeClaimTemplate)
	}
	if cluster.Spec.SecurityContext != nil {
		ss.Spec.Template.Spec.SecurityContext = cluster.Spec.SecurityContext
	}
	if cluster.Spec.Tolerations != nil {
		ss.Spec.Template.Spec.Tolerations = *cluster.Spec.Tolerations
	}
	return ss
}
