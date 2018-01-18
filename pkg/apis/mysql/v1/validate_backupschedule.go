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
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func validateBackupSchedule(bs *MySQLBackupSchedule) field.ErrorList {
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
