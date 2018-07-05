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

	"github.com/coreos/go-semver/semver"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/oracle/mysql-operator/pkg/constants"
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
	allErrs = append(allErrs, validateMembers(s.Members, fldPath.Child("members"))...)
	allErrs = append(allErrs, validateBaseServerID(s.BaseServerID, fldPath.Child("baseServerId"))...)

	return allErrs
}

func validateClusterStatus(s ClusterStatus, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	return allErrs
}

func validateVersion(version string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	min, err := semver.NewVersion(MinimumMySQLVersion)
	if err != nil {
		allErrs = append(allErrs, field.InternalError(fldPath, fmt.Errorf("unable to parse minimum MySQL version: %v", err)))
	}

	given, err := semver.NewVersion(version)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(fldPath, version, fmt.Sprintf("unable to parse MySQL version: %v", err)))
	}

	if len(allErrs) == 0 {
		if given.Compare(*min) == -1 {
			allErrs = append(allErrs, field.Invalid(fldPath, version, fmt.Sprintf("minimum supported MySQL version is %s", MinimumMySQLVersion)))
		}
	}

	return allErrs
}

func validateBaseServerID(baseServerID uint32, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if baseServerID <= maxBaseServerID {
		return allErrs
	}
	return append(allErrs, field.Invalid(fldPath, strconv.FormatUint(uint64(baseServerID), 10), "invalid baseServerId specified"))
}

func validateMembers(members int32, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if members < 1 || members > MaxInnoDBClusterMembers {
		allErrs = append(allErrs, field.Invalid(fldPath, members, "InnoDB clustering supports between 1-9 members"))
	}
	return allErrs
}

func validateS3StorageProvider(s3 *S3StorageProvider, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if s3.Region == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("region"), ""))
	}
	if s3.Endpoint == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("endpoint"), ""))
	}
	if s3.Bucket == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("bucket"), ""))
	}

	if s3.CredentialsSecret == nil {
		allErrs = append(allErrs, field.Required(fldPath.Child("credentialsSecret"), ""))
	} else if s3.CredentialsSecret.Name == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("credentialsSecret").Child("name"), ""))
	}

	return allErrs
}

func validateStorageProvider(storage StorageProvider, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if storage.S3 == nil {
		allErrs = append(allErrs, field.Required(fldPath.Child("s3"), "S3 (compatible) is currently the only supported storage provider"))
	} else {
		allErrs = append(allErrs, validateS3StorageProvider(storage.S3, fldPath.Child("s3"))...)
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

func validateDatabase(database Database, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if database.Name == "" {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("name"), database.Name, ""))
	}

	return allErrs
}

func validateMySQLDumpExecutor(executor *MySQLDumpBackupExecutor, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if executor.Databases == nil || len(executor.Databases) == 0 {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("databases"), executor, ""))
	}

	for i, database := range executor.Databases {
		allErrs = append(allErrs, validateDatabase(database, fldPath.Index(i))...)
	}

	return allErrs
}

func validateExecutor(executor BackupExecutor, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if executor.MySQLDump == nil {
		allErrs = append(allErrs, field.Required(fldPath.Child("mysqldump"), "mysqldump is currently the only supported backup mechanism"))
	} else {
		allErrs = append(allErrs, validateMySQLDumpExecutor(executor.MySQLDump, fldPath.Child("mysqldump"))...)
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

	allErrs = append(allErrs, validateExecutor(spec.Executor, field.NewPath("executor"))...)
	allErrs = append(allErrs, validateStorageProvider(spec.StorageProvider, field.NewPath("storageProvider"))...)

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
