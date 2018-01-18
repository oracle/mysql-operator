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

// +build all default

package e2e

import (
	"testing"

	"github.com/oracle/mysql-operator/test/e2e/framework"
	e2eutil "github.com/oracle/mysql-operator/test/e2e/util"
)

// TestCrashRecovery spins up a 3-instance cluster then checks then check the
// various pod based failure modes
func TestPodCrashRecovery(test *testing.T) {
	t := e2eutil.NewT(test)
	f := framework.Global
	namespace := f.Namespace
	var numInstances int32 = 3
	var testdb *e2eutil.TestDB

	testdb = e2eutil.CreateTestDB(t, "e2e-pr-", numInstances, false, f.DestroyAfterFailure)
	defer testdb.Delete()

	t.Log("=============== Populating the database ===============")
	testdb.Populate()
	t.Log("=============== Validating the database ===============")
	testdb.Test()
	t.Log("--------------- Complete ---------------")

	clusterName := testdb.GetClusterName()
	var podName string

	t.Log("=============== Testing mysql primary pod crash ===============")
	podName = e2eutil.GetPrimaryPodName(t, namespace, clusterName, f.KubeClient)
	e2eutil.TestMySQLPodCrash(t, namespace, podName, f.KubeClient, clusterName, numInstances)
	e2eutil.CheckPrimaryFailover(t, namespace, clusterName, podName, f.KubeClient)
	t.Log("--------------- Test complete ---------------")

	t.Log("=============== Testing mysql secondary pod crash ===============")
	podName = e2eutil.GetSecondaryPodName(t, namespace, clusterName, f.KubeClient)
	e2eutil.TestMySQLPodCrash(t, namespace, podName, f.KubeClient, clusterName, numInstances)
	t.Log("--------------- Test complete ---------------")

	t.Log("=============== Validating the database ===============")
	testdb.Test()
	t.Log("--------------- Complete ---------------")

	t.Report()
}
