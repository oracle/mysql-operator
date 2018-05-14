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

package e2e

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/oracle/mysql-operator/test/e2e/framework"
)

var _ = Describe("Basic cluster creation", func() {
	f := framework.NewDefaultFramework("basic")

	It("should be possible to create a basic 3 member cluster with a 28 character name", func() {
		clusterName := "basic-twenty-eight-char-name"
		Expect(clusterName).To(HaveLen(28))

		jig := framework.NewMySQLClusterTestJig(f.MySQLClientSet, f.ClientSet, clusterName)

		cluster := jig.CreateAndAwaitMySQLClusterOrFail(f.Namespace.Name, 3, nil, framework.DefaultTimeout)
		framework.RWSQLTest(cluster, cluster.Name+"-0")
	})
})
