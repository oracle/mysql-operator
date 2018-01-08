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
