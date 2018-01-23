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

package fake

import (
	internalversion "github.com/oracle/mysql-operator/pkg/generated/clientset/internalversion/typed/mysql/internalversion"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeMysql struct {
	*testing.Fake
}

func (c *FakeMysql) MySQLBackups(namespace string) internalversion.MySQLBackupInterface {
	return &FakeMySQLBackups{c, namespace}
}

func (c *FakeMysql) MySQLBackupSchedules(namespace string) internalversion.MySQLBackupScheduleInterface {
	return &FakeMySQLBackupSchedules{c, namespace}
}

func (c *FakeMysql) MySQLClusters(namespace string) internalversion.MySQLClusterInterface {
	return &FakeMySQLClusters{c, namespace}
}

func (c *FakeMysql) MySQLRestores(namespace string) internalversion.MySQLRestoreInterface {
	return &FakeMySQLRestores{c, namespace}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeMysql) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
