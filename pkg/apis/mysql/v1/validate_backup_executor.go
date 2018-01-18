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
)

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

func validateExecutor(executor *Executor, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if executor.Provider == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("provider"), ""))
	} else if !isValidExecutorProvider(executor.Provider) {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("provider"), executor, fmt.Sprintf("invalid provider name '%s'", executor.Provider)))
	}

	if executor.Databases == nil || len(executor.Databases) == 0 {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("databases"), executor, "missing databases"))
	}

	return allErrs
}
