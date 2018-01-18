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

package s3

import (
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/oracle/mysql-operator/pkg/apis/mysql/v1"
)

// Config holds the credentials required to authenticate with an S3 compliant API.
type Config struct {
	endpoint  string
	region    string
	bucket    string
	accessKey string
	secretKey string
}

// NewConfig creates an S3 configuration based on the input parameters.
func NewConfig(storage *v1.Storage, creds map[string]string) *Config {
	return &Config{
		endpoint:  storage.Config["endpoint"],
		region:    storage.Config["region"],
		bucket:    storage.Config["bucket"],
		accessKey: creds["accessKey"],
		secretKey: creds["secretKey"],
	}
}

// Validate checks the required S3 configuration parameters are set.
func (c *Config) Validate() error {
	allErrs := field.ErrorList{}
	fldPath := field.NewPath("data")

	if c.endpoint == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("endpoint"), ""))
	}
	if c.region == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("region"), ""))
	}
	if c.bucket == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("bucket"), ""))
	}
	if c.accessKey == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("accessKey"), ""))
	}
	if c.secretKey == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("secretKey"), ""))
	}

	return allErrs.ToAggregate()
}
