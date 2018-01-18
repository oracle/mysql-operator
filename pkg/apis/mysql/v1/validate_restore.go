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
	"fmt"

	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/oracle/mysql-operator/pkg/constants"
)

func validateRestore(restore *MySQLRestore) field.ErrorList {
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

	if s.ClusterRef == nil || s.ClusterRef.Name == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("clusterRef").Child("name"), "a cluster to restore into is required"))
	}

	if s.BackupRef == nil || s.BackupRef.Name == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("backupRef").Child("name"), "a backup to restore is required"))
	}

	return allErrs
}
