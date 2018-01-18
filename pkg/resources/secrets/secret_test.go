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

package secrets

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	api "github.com/oracle/mysql-operator/pkg/apis/mysql/v1"
)

func TestGetRootPasswordSecretName(t *testing.T) {
	cluster := &api.MySQLCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "example-cluster"},
		Spec:       api.MySQLClusterSpec{},
	}

	actual := GetRootPasswordSecretName(cluster)

	if actual != "example-cluster-root-password" {
		t.Errorf("Expected example-cluster-root-password but got %s", actual)
	}
}
