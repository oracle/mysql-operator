package v1

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation/field"
)

const (
	// ProviderS3 denotes S3 compatability backed storage provider.
	ProviderS3 = "s3"
)

func validateStorage(storage *Storage, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if storage.Provider == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("provider"), ""))
	}

	if storage.Config == nil {
		allErrs = append(allErrs, field.Required(fldPath.Child("config"), ""))
	} else {
		switch strings.ToLower(storage.Provider) {
		case ProviderS3:
			allErrs = append(allErrs, validateS3StorageConfig(storage.Config, field.NewPath("config"))...)
		default:
			allErrs = append(allErrs, field.Invalid(fldPath.Child("provider"), storage, fmt.Sprintf("invalid storage name '%s'. Permitted names: s3.", storage.Provider)))
		}
	}

	if storage.SecretRef == nil {
		allErrs = append(allErrs, field.Required(fldPath.Child("secretRef"), ""))
	} else if storage.SecretRef.Name == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("secretRef").Child("name"), ""))
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
