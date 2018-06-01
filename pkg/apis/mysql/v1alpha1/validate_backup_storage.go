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
	"strings"

	"k8s.io/apimachinery/pkg/util/validation/field"
)

const (
	// ProviderS3 denotes S3 compatability backed storage provider.
	ProviderS3 = "s3"
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
		case ProviderS3:
			allErrs = append(allErrs, validateS3StorageConfig(storage.Config, field.NewPath("config"))...)
		default:
			allErrs = append(allErrs, field.Invalid(fldPath.Child("name"), storage, fmt.Sprintf("invalid storage name '%s'. Permitted names: s3.", storage.Name)))
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
