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

package util

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	api "github.com/oracle/mysql-operator/pkg/apis/mysql/v1"
	mysqlop "github.com/oracle/mysql-operator/pkg/generated/clientset/versioned"
)

func CreateMySQLCluster(t *testing.T, mysqlopClient mysqlop.Interface, ns string, cluster *api.MySQLCluster) (*api.MySQLCluster, error) {
	cluster.Namespace = ns
	res, err := mysqlopClient.MysqlV1().MySQLClusters(ns).Create(cluster)
	if err != nil {
		return nil, err
	}
	t.Logf("Creating mysql cluster: %s", res.Name)
	return res, nil
}

// TODO(apryde): Wait for deletion of underlying resources.
func DeleteMySQLCluster(t *testing.T, mysqlopClient mysqlop.Interface, cluster *api.MySQLCluster) error {
	t.Logf("Deleting mysql cluster: %s", cluster.Name)
	err := mysqlopClient.MysqlV1().MySQLClusters(cluster.Namespace).Delete(cluster.Name, &metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	return nil
}
