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
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func TestValidateVersion(t *testing.T) {
	fldPath := field.NewPath("spec", "version")
	testCases := map[string]struct {
		name     string
		version  string
		expected field.ErrorList
	}{
		"minimum_version_valid": {
			version:  MinimumMySQLVersion,
			expected: field.ErrorList{},
		},
		"next_patch_version_valid": {
			version:  "8.0.12",
			expected: field.ErrorList{},
		},
		"next_minor_version_valid": {
			version:  "8.1.0",
			expected: field.ErrorList{},
		},
		"previous_version_invalid": {
			version: "8.0.4",
			expected: field.ErrorList{
				field.Invalid(fldPath, "8.0.4", fmt.Sprintf("minimum supported MySQL version is %s", MinimumMySQLVersion)),
			},
		},
		"5.7_version_invalid": {
			version: "5.7.20-1.1.2",
			expected: field.ErrorList{
				field.Invalid(fldPath, "5.7.20-1.1.2", fmt.Sprintf("minimum supported MySQL version is %s", MinimumMySQLVersion)),
			},
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			errs := validateVersion(tc.version, fldPath)
			assert.EqualValues(t, errs, tc.expected)
		})
	}
}

func TestDefaultMembers(t *testing.T) {
	cluster := &Cluster{}
	cluster.EnsureDefaults()

	if cluster.Spec.Members != defaultMembers {
		t.Errorf("Expected default members to be %d but got %d", defaultMembers, cluster.Spec.Members)
	}
}

func TestDefaultBaseServerID(t *testing.T) {
	cluster := &Cluster{}
	cluster.EnsureDefaults()

	if cluster.Spec.BaseServerID != defaultBaseServerID {
		t.Errorf("Expected default BaseServerID to be %d but got %d", defaultBaseServerID, cluster.Spec.BaseServerID)
	}
}

func TestDefaultVersion(t *testing.T) {
	cluster := &Cluster{}
	cluster.EnsureDefaults()

	if cluster.Spec.Version != DefaultVersion {
		t.Errorf("Expected default version to be %s but got %s", DefaultVersion, cluster.Spec.Version)
	}
}

func TestRequiresConfigMount(t *testing.T) {
	cluster := &Cluster{}
	cluster.EnsureDefaults()

	if cluster.RequiresConfigMount() {
		t.Errorf("Cluster without config should not require a config mount")
	}

	cluster = &Cluster{
		Spec: ClusterSpec{
			Config: &corev1.LocalObjectReference{
				Name: "customconfig",
			},
		},
	}

	if !cluster.RequiresConfigMount() {
		t.Errorf("Cluster with config should require a config mount")
	}
}

func TestRequiresCustomSSLSetup(t *testing.T) {
	cluster := &Cluster{}
	cluster.EnsureDefaults()

	assert.False(t, cluster.RequiresCustomSSLSetup(), "Cluster without SSLSecret should not require custom SSL setup")

	cluster = &Cluster{
		Spec: ClusterSpec{
			SSLSecret: &corev1.LocalObjectReference{
				Name: "custom-ssl-secret",
			},
		},
	}

	assert.True(t, cluster.RequiresCustomSSLSetup(), "Cluster with SSLSecret should require custom SSL setup")
}
