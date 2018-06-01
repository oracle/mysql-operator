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

package cluster

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/oracle/mysql-operator/pkg/apis/mysql/v1alpha1"
	"github.com/oracle/mysql-operator/pkg/resources/secrets"
)

// SecretControlInterface defines the interface that the ClusterController
// uses to get and create Secrets. It is implemented as an interface to enable
// testing.
type SecretControlInterface interface {
	GetForCluster(cluster *v1alpha1.Cluster) (*v1.Secret, error)
	CreateSecret(s *v1.Secret) error
}

type realSecretControl struct {
	client kubernetes.Interface
}

// NewRealSecretControl creates a concrete implementation of the
// SecretControlInterface.
func NewRealSecretControl(client kubernetes.Interface) SecretControlInterface {
	return &realSecretControl{client: client}
}

func (rsc *realSecretControl) GetForCluster(cluster *v1alpha1.Cluster) (*v1.Secret, error) {
	return rsc.client.CoreV1().
		Secrets(cluster.Namespace).
		Get(secrets.GetRootPasswordSecretName(cluster), metav1.GetOptions{})
}

func (rsc *realSecretControl) CreateSecret(s *v1.Secret) error {
	_, err := rsc.client.CoreV1().Secrets(s.Namespace).Create(s)
	return err
}
