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

	apps "k8s.io/api/apps/v1beta1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"

	agentopts "github.com/oracle/mysql-operator/cmd/mysql-agent/app/options"
	operatoropts "github.com/oracle/mysql-operator/cmd/mysql-operator/app/options"
	api "github.com/oracle/mysql-operator/pkg/apis/mysql/v1"
	"github.com/oracle/mysql-operator/pkg/constants"
	"github.com/oracle/mysql-operator/pkg/resources/secrets"
	"github.com/oracle/mysql-operator/pkg/version"
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

	replicationGroupPort = 13306
)

func volumeMounts(cluster *api.MySQLCluster) []v1.VolumeMount {
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

	return mounts
}

func clusterNameEnvVar(cluster *api.MySQLCluster) v1.EnvVar {
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
func mysqlRootPassword(cluster *api.MySQLCluster) v1.EnvVar {
	var secretName string
	if cluster.RequiresSecret() {
		secretName = secrets.GetRootPasswordSecretName(cluster)
	} else {
		secretName = cluster.Spec.SecretRef.Name
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

func serviceNameEnvVar(serviceName string) v1.EnvVar {
	return v1.EnvVar{
		Name:  "MYSQL_CLUSTER_SERVICE_NAME",
		Value: serviceName,
	}
}

func getReplicationGroupSeeds(serviceName string, replicas int) string {
	seeds := []string{}
	for i := 0; i < replicas; i++ {
		seeds = append(seeds, fmt.Sprintf("%s-%d:%d",
			serviceName, i, replicationGroupPort))
	}
	return strings.Join(seeds, ",")
}

// Builds the MySQL operator container for a cluster.
// The 'mysqlImage' parameter is the image name of the mysql server to use with
// no version information.. e.g. 'mysql/mysql-server'
func mysqlServerContainer(cluster *api.MySQLCluster, mysqlServerImage string, rootPassword v1.EnvVar, serviceName string, replicas int) v1.Container {
	replicationGroupSeeds := getReplicationGroupSeeds(serviceName, replicas)

	args := []string{
		"--server_id=$(expr 1000 + $index)",
		// basic process setup options
		"--user=mysql",
		"--datadir=/var/lib/mysql",
		// storage engine options
		"--default-storage-engine=innodb",
		"--default-tmp-storage-engine=innodb",
		"--internal-tmp-disk-storage-engine=innodb",
		// character set, collation, and i18n options
		"--character-set-server=utf8mb4",
		"--collation-server=utf8mb4_unicode_520_ci",
		// crash handling and debugging options
		"--core-file",
		"--default-password-lifetime=0",
		// date and time handling options
		"--default-time-zone=SYSTEM",
		"--explicit-defaults-for-timestamp=ON",
		// performance Schema options
		"--performance-schema-consumer-events-transactions-current=ON",
		"--performance-schema-consumer-events-transactions-history=ON",
		// innoDB options
		"--innodb-buffer-pool-size=128M",
		"--innodb-buffer-pool-instances=4",
		"--innodb-autoinc-lock-mode=2",
		"--innodb-flush-method=O_DIRECT_NO_FSYNC",
		"--innodb-open-files=128",
		"--innodb-log-buffer-size=4M",
		"--innodb-monitor-enable='%'",
		"--innodb-print-all-deadlocks=ON",
		"--innodb-undo-log-truncate=ON",
		"--innodb-undo-tablespaces=2",
		"--innodb-undo-logs=2",
		// group replication pre-requisites & recommendations
		"--binlog_checksum=NONE",
		"--gtid_mode=ON",
		"--enforce_gtid_consistency=ON",
		"--log_bin",
		"--binlog-format=ROW",
		"--log-slave-updates=ON",
		"--master-info-repository=TABLE",
		"--relay-log-info-repository=TABLE",
		"--slave-preserve-commit-order=ON",
		"--disabled_storage_engines='MyISAM,BLACKHOLE,FEDERATED,ARCHIVE'",
		"--transaction-isolation='READ-COMMITTED'",
		// group replication specific options
		"--transaction-write-set-extraction=XXHASH64",
		"--loose-group-replication-ip-whitelist='0.0.0.0/0'",
	}

	entryPointArgs := strings.Join(args, " ")

	cmd := fmt.Sprintf(
		`# Note: We fiddle with the resolv.conf file in order to ensure that the mysql instances
         # can refer to each other using just thier hostnames (e.g. mysql-N), thus do not need
         # to qualify their names with the name of the (headless) service (e.g. mysql-N.mysql)
         search=$(grep ^search /etc/resolv.conf)
         echo "$search %s.${POD_NAMESPACE}.svc.cluster.local" >> /etc/resolv.conf

         # Finds the replica index from the hostname, and uses this to define
         # a unique server id for this instance.
         index=$(cat /etc/hostname | grep -o '[^-]*$')
         /entrypoint.sh %s`,
		serviceName, entryPointArgs)

	return v1.Container{
		Name: MySQLServerName,
		// TODO(apryde): Add BaseImage to cluster CRD.
		Image: fmt.Sprintf("%s:%s", mysqlServerImage, cluster.Spec.Version),
		Ports: []v1.ContainerPort{
			v1.ContainerPort{
				ContainerPort: 3306,
			},
		},
		VolumeMounts: volumeMounts(cluster),
		Command:      []string{"/bin/bash", "-ecx", cmd},
		Env: []v1.EnvVar{
			clusterNameEnvVar(cluster),
			namespaceEnvVar(),
			serviceNameEnvVar(serviceName),
			replicationGroupSeedsEnvVar(replicationGroupSeeds),
			multiMasterEnvVar(cluster.Spec.MultiMaster),
			rootPassword,
			v1.EnvVar{
				Name:  "MYSQL_ROOT_HOST",
				Value: "%",
			},
			v1.EnvVar{
				Name:  "MYSQL_LOG_CONSOLE",
				Value: "true",
			},
		},
	}
}

func mysqlAgentContainer(cluster *api.MySQLCluster, mysqlAgentImage string, rootPassword v1.EnvVar, serviceName string, replicas int) v1.Container {
	agentVersion := version.GetBuildVersion()
	if version := os.Getenv("MYSQL_AGENT_VERSION"); version != "" {
		agentVersion = version
	}

	replicationGroupSeeds := getReplicationGroupSeeds(serviceName, replicas)

	return v1.Container{
		Name:         MySQLAgentName,
		Image:        fmt.Sprintf("%s:%s", mysqlAgentImage, agentVersion),
		Args:         []string{"--v=4"},
		VolumeMounts: volumeMounts(cluster),
		Env: []v1.EnvVar{
			clusterNameEnvVar(cluster),
			namespaceEnvVar(),
			serviceNameEnvVar(serviceName),
			replicationGroupSeedsEnvVar(replicationGroupSeeds),
			multiMasterEnvVar(cluster.Spec.MultiMaster),
			rootPassword,
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
	}
}

// NewForCluster creates a new StatefulSet for the given MySQLCluster.
func NewForCluster(cluster *api.MySQLCluster, images operatoropts.Images, serviceName string) *apps.StatefulSet {
	rootPassword := mysqlRootPassword(cluster)
	replicas := int(cluster.Spec.Replicas)

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
						Name: cluster.Spec.ConfigRef.Name,
					},
				},
			},
		})
	}

	containers := []v1.Container{
		mysqlServerContainer(cluster, images.MySQLServerImage, rootPassword, serviceName, replicas),
		mysqlAgentContainer(cluster, images.MySQLAgentImage, rootPassword, serviceName, replicas)}

	podLabels := map[string]string{
		constants.MySQLClusterLabel: cluster.Name,
	}
	if cluster.Spec.MultiMaster {
		podLabels[constants.LabelMySQLClusterRole] = constants.MySQLClusterRolePrimary
	}

	ss := &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: cluster.Namespace,
			Name:      cluster.Name,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(cluster, schema.GroupVersionKind{
					Group:   api.SchemeGroupVersion.Group,
					Version: api.SchemeGroupVersion.Version,
					Kind:    api.MySQLClusterCRDResourceKind,
				}),
			},
			Labels: map[string]string{
				constants.MySQLClusterLabel:         cluster.Name,
				constants.MySQLOperatorVersionLabel: version.GetBuildVersion(),
			},
		},
		Spec: apps.StatefulSetSpec{
			Replicas: &cluster.Spec.Replicas,
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
			ServiceName: serviceName,
		},
	}

	if cluster.Spec.VolumeClaimTemplate != nil {
		ss.Spec.VolumeClaimTemplates = append(ss.Spec.VolumeClaimTemplates, *cluster.Spec.VolumeClaimTemplate)
	}
	if cluster.Spec.BackupVolumeClaimTemplate != nil {
		ss.Spec.VolumeClaimTemplates = append(ss.Spec.VolumeClaimTemplates, *cluster.Spec.BackupVolumeClaimTemplate)
	}
	return ss
}
