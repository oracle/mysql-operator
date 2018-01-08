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
