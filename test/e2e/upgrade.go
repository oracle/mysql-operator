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

	"github.com/oracle/mysql-operator/pkg/apis/mysql/v1alpha1"
	mysqlclientset "github.com/oracle/mysql-operator/pkg/generated/clientset/versioned"
	"github.com/oracle/mysql-operator/test/e2e/framework"
)

var _ = Describe("MySQL Upgrade", func() {
	f := framework.NewDefaultFramework("upgrade")

	var mcs mysqlclientset.Interface
	BeforeEach(func() {
		mcs = f.MySQLClientSet
	})

	It("should be possible to upgrade a cluster from 8.0.11 to 8.0.12", func() {
		jig := framework.NewClusterTestJig(mcs, f.ClientSet, "upgrade-test")

		By("creating an 8.0.11 cluster")

		cluster := jig.CreateAndAwaitClusterOrFail(f.Namespace.Name, 3, func(c *v1alpha1.Cluster) {
			c.Spec.Version = "8.0.11"
		}, framework.DefaultTimeout)

		expected, err := framework.WriteSQLTest(cluster, cluster.Name+"-0")
		Expect(err).NotTo(HaveOccurred())

		By("triggering an upgrade to 8.0.12")

		cluster.Spec.Version = "8.0.12"
		cluster, err = mcs.MySQLV1alpha1().Clusters(cluster.Namespace).Update(cluster)
		Expect(err).NotTo(HaveOccurred())

		By("waiting for the upgrade to complete")

		cluster = jig.WaitForClusterUpgradedOrFail(cluster.Namespace, cluster.Name, "8.0.12", framework.DefaultTimeout)

		By("testing we can read from the upgraded database")

		actual, err := framework.ReadSQLTest(cluster, cluster.Name+"-0")
		Expect(err).NotTo(HaveOccurred())
		Expect(actual).To(Equal(expected))
	})
})
