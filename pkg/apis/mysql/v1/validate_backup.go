package v1

import (
	"fmt"

	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/oracle/mysql-operator/pkg/constants"
)

func validateBackup(backup *MySQLBackup) field.ErrorList {
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

	if spec.Storage == nil {
		allErrs = append(allErrs, field.Required(fldPath.Child("storage"), "missing storage"))
	} else {
		allErrs = append(allErrs, validateStorage(spec.Storage, field.NewPath("storage"))...)
	}

	if spec.ClusterRef == nil {
		allErrs = append(allErrs, field.Required(fldPath.Child("clusterRef"), "missing cluster"))
	}

	return allErrs
}
