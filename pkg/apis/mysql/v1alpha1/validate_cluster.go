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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func validateCluster(c *MySQLCluster) field.ErrorList {
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

	if len(name) > MySQLClusterNameMaxLen {
		msg := fmt.Sprintf("longer than maximum supported length %d (see: https://bugs.mysql.com/bug.php?id=90601)", MaxInnoDBClusterMembers)
		allErrs = append(allErrs, field.Invalid(fldPath, name, msg))
	}

	return allErrs
}

func validateClusterSpec(s MySQLClusterSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, validateVersion(s.Version, fldPath.Child("version"))...)
	allErrs = append(allErrs, validateReplicas(s.Replicas, fldPath.Child("replicas"))...)
	allErrs = append(allErrs, validateBaseServerID(s.BaseServerID, fldPath.Child("baseServerId"))...)

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

func validateBaseServerID(baseServerID uint32, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if baseServerID <= maxBaseServerID {
		return allErrs
	}
	return append(allErrs, field.Invalid(fldPath, strconv.FormatUint(uint64(baseServerID), 10), "invalid baseServerId specified"))
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

func validateReplicas(replicas int32, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if replicas < 1 || replicas > MaxInnoDBClusterMembers {
		allErrs = append(allErrs, field.Invalid(fldPath, replicas, "InnoDB clustering supports between 1-9 members"))
	}
	return allErrs
}
