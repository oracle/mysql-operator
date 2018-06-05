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

package v1alpha1

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/oracle/mysql-operator/pkg/constants"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func validateCluster(c *Cluster) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, validateClusterMetadata(c.ObjectMeta, field.NewPath("metadata"))...)
	allErrs = append(allErrs, validateClusterSpec(c.Spec, field.NewPath("spec"))...)
	allErrs = append(allErrs, validateClusterStatus(c.Status, field.NewPath("status"))...)
	return allErrs
}

func validateClusterMetadata(m metav1.ObjectMeta, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, validateName(m.Name, fldPath.Child("name"))...)

	return allErrs
}

func validateName(name string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if len(name) > ClusterNameMaxLen {
		msg := fmt.Sprintf("longer than maximum supported length %d (see: https://bugs.mysql.com/bug.php?id=90601)", ClusterNameMaxLen)
		allErrs = append(allErrs, field.Invalid(fldPath, name, msg))
	}

	return allErrs
}

func validateClusterSpec(s ClusterSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, validateVersion(s.Version, fldPath.Child("version"))...)
	allErrs = append(allErrs, validateReplicas(s.Replicas, fldPath.Child("replicas"))...)
	allErrs = append(allErrs, validateBaseServerID(s.BaseServerID, fldPath.Child("baseServerId"))...)

	return allErrs
}

func validateClusterStatus(s ClusterStatus, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	return allErrs
}

func validateVersion(version string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	for _, validVersion := range validVersions {
		if version == validVersion {
			return allErrs
		}
	}
	return append(allErrs, field.Invalid(fldPath, version, "invalid version specified"))
}

func validateBaseServerID(baseServerID uint32, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if baseServerID <= maxBaseServerID {
		return allErrs
	}
	return append(allErrs, field.Invalid(fldPath, strconv.FormatUint(uint64(baseServerID), 10), "invalid baseServerId specified"))
}

func validateReplicas(replicas int32, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if replicas < 1 || replicas > MaxInnoDBClusterMembers {
		allErrs = append(allErrs, field.Invalid(fldPath, replicas, "InnoDB clustering supports between 1-9 members"))
	}
	return allErrs
}

const (
	// ProviderNameS3 denotes S3 compatability backed storage provider.
	ProviderNameS3 = "s3"
)

func validateStorage(storage *BackupStorageProvider, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if storage.Name == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("name"), ""))
	}

	if storage.Config == nil {
		allErrs = append(allErrs, field.Required(fldPath.Child("config"), ""))
	} else {
		switch strings.ToLower(storage.Name) {
		case ProviderNameS3:
			allErrs = append(allErrs, validateS3StorageConfig(storage.Config, field.NewPath("config"))...)
		default:
			allErrs = append(allErrs, field.Invalid(fldPath.Child("name"), storage, fmt.Sprintf("invalid storage name '%s'. Permitted names: s3.", storage.Name)))
		}
	}

	if storage.AuthSecret == nil {
		allErrs = append(allErrs, field.Required(fldPath.Child("authSecret"), ""))
	} else if storage.AuthSecret.Name == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("authSecret").Child("name"), ""))
	}

	return allErrs
}

func validateS3StorageConfig(config map[string]string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if config["endpoint"] == "" {
		allErrs = append(allErrs, field.Required(fldPath.Key("endpoint"), "missing S3 storage config 'endpoint' value"))
	}

	if config["region"] == "" {
		allErrs = append(allErrs, field.Required(fldPath.Key("region"), "missing S3 storage config 'region' value"))
	}

	if config["bucket"] == "" {
		allErrs = append(allErrs, field.Required(fldPath.Key("bucket"), "missing S3 storage config 'bucket' value"))
	}

	return allErrs
}

// ExecutorProviders denotes the list of valid backup executor providers.
var ExecutorProviders = []string{"mysqldump"}

func isValidExecutorProvider(provider string) bool {
	for _, ex := range ExecutorProviders {
		if provider == ex {
			return true
		}
	}

	return false
}

func validateExecutor(executor *BackupExecutor, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if executor.Name == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("name"), ""))
	} else if !isValidExecutorProvider(executor.Name) {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("name"), executor, fmt.Sprintf("invalid provider name %q", executor.Name)))
	}

	if executor.Databases == nil || len(executor.Databases) == 0 {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("databases"), executor, "missing databases"))
	}

	return allErrs
}

func validateBackup(backup *Backup) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, validateBackupLabels(backup.Labels, field.NewPath("labels"))...)
	allErrs = append(allErrs, validateBackupSpec(backup.Spec, field.NewPath("spec"))...)

	return allErrs
}

func validateBackupLabels(labels map[string]string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if labels[constants.MySQLOperatorVersionLabel] == "" {
		errorStr := fmt.Sprintf("no '%s' present.", constants.MySQLOperatorVersionLabel)
		allErrs = append(allErrs, field.Invalid(fldPath, labels, errorStr))
	}
	return allErrs
}

func validateBackupSpec(spec BackupSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if spec.Executor == nil {
		allErrs = append(allErrs, field.Required(fldPath.Child("executor"), "missing executor"))
	} else {
		allErrs = append(allErrs, validateExecutor(spec.Executor, field.NewPath("executor"))...)
	}

	if spec.StorageProvider == nil {
		allErrs = append(allErrs, field.Required(fldPath.Child("storageProvider"), "missing storage provider"))
	} else {
		allErrs = append(allErrs, validateStorage(spec.StorageProvider, field.NewPath("storageProvider"))...)
	}

	if spec.Cluster == nil {
		allErrs = append(allErrs, field.Required(fldPath.Child("cluster"), "missing cluster"))
	}

	return allErrs
}

func validateBackupSchedule(bs *BackupSchedule) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, validateBackupScheduleSpec(bs.Spec, field.NewPath("spec"))...)
	return allErrs
}

func validateBackupScheduleSpec(spec BackupScheduleSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if &spec.BackupTemplate == nil {
		allErrs = append(allErrs, field.Required(fldPath.Child("backupTemplate"), "missing backup template"))
	} else {
		allErrs = append(allErrs, validateBackupSpec(spec.BackupTemplate, field.NewPath("backupTemplate"))...)
	}

	return allErrs
}

func validateRestore(restore *Restore) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, validateRestoreSpec(restore.Spec, field.NewPath("spec"))...)

	// FIXME(apryde): The version label is a piece of internal bookkeeping,
	// however, we're validating with a user-facing error here. Should test
	// we're applying the label in unit tests and remove this validation.
	value, ok := restore.Labels[constants.MySQLOperatorVersionLabel]
	if !ok {
		errorStr := fmt.Sprintf("no '%s' present.", constants.MySQLOperatorVersionLabel)
		allErrs = append(allErrs, field.Invalid(field.NewPath("labels"), restore.Labels, errorStr))
	}
	if value == "" {
		errorStr := fmt.Sprintf("empty '%s' present.", constants.MySQLOperatorVersionLabel)
		allErrs = append(allErrs, field.Invalid(field.NewPath("labels"), restore.Labels, errorStr))
	}

	return allErrs
}

func validateRestoreSpec(s RestoreSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if s.Cluster == nil || s.Cluster.Name == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("cluster").Child("name"), "a cluster to restore into is required"))
	}

	if s.Backup == nil || s.Backup.Name == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("backup").Child("name"), "a backup to restore is required"))
	}

	return allErrs
}
