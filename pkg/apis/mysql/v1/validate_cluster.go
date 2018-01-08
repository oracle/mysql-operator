package v1

import (
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func validateCluster(c *MySQLCluster) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, validateClusterSpec(c.Spec, field.NewPath("spec"))...)
	allErrs = append(allErrs, validateClusterStatus(c.Status, field.NewPath("status"))...)
	return allErrs
}

func validateClusterSpec(s MySQLClusterSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, validateVersion(s.Version, fldPath.Child("version"))...)

	return allErrs
}

func validateClusterStatus(s MySQLClusterStatus, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, validatePhase(s.Phase, fldPath.Child("phase"))...)
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

func validatePhase(phase MySQLClusterPhase, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	for _, validPhase := range MySQLClusterValidPhases {
		if phase == validPhase {
			return allErrs
		}
	}
	return append(allErrs, field.Invalid(fldPath, phase, "invalid phase specified"))
}
