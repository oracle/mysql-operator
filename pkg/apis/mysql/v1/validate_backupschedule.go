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
