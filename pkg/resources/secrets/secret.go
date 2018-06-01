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
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/oracle/mysql-operator/pkg/apis/mysql/v1alpha1"
	"github.com/oracle/mysql-operator/pkg/constants"
)

// NewMysqlRootPassword returns a Kubernetes secret containing a
// generated MySQL root password.
func NewMysqlRootPassword(cluster *v1alpha1.Cluster) *corev1.Secret {
	CreateSecret := RandomAlphanumericString(16)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{constants.ClusterLabel: cluster.Name},
			Name:   GetRootPasswordSecretName(cluster),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(cluster, schema.GroupVersionKind{
					Group:   v1alpha1.SchemeGroupVersion.Group,
					Version: v1alpha1.SchemeGroupVersion.Version,
					Kind:    v1alpha1.ClusterCRDResourceKind,
				}),
			},
			Namespace: cluster.Namespace,
		},
		Data: map[string][]byte{"password": []byte(CreateSecret)},
	}
	return secret
}

// GetRootPasswordSecretName returns the root password secret name for the
// given mysql cluster.
func GetRootPasswordSecretName(cluster *v1alpha1.Cluster) string {
	return fmt.Sprintf("%s-root-password", cluster.Name)
}
