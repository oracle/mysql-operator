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
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/oracle/mysql-operator/pkg/apis/mysql/v1alpha1"
	"github.com/oracle/mysql-operator/test/e2e/framework"
)

var _ = Describe("Cluster creation", func() {
	f := framework.NewDefaultFramework("cluster-creation")

	It("should be possible to create a basic 3 member cluster with a 28 character name", func() {
		clusterName := "basic-twenty-eight-char-name"
		Expect(clusterName).To(HaveLen(28))

		jig := framework.NewClusterTestJig(f.MySQLClientSet, f.ClientSet, clusterName)

		cluster := jig.CreateAndAwaitClusterOrFail(f.Namespace.Name, 3, nil, framework.DefaultTimeout)

		expected, err := framework.WriteSQLTest(cluster, cluster.Name+"-0")
		Expect(err).NotTo(HaveOccurred())

		actual, err := framework.ReadSQLTest(cluster, cluster.Name+"-0")
		Expect(err).NotTo(HaveOccurred())
		Expect(actual).To(Equal(expected))
	})

	It("should be possible to create a multi-master cluster", func() {
		clusterName := "multi-master"
		members := int32(3)

		jig := framework.NewClusterTestJig(f.MySQLClientSet, f.ClientSet, clusterName)

		cluster := jig.CreateAndAwaitClusterOrFail(f.Namespace.Name, members, func(cluster *v1alpha1.Cluster) {
			cluster.Spec.MultiMaster = true
		}, framework.DefaultTimeout)

		By("Checking we can write to and read from to all members")
		for i := int32(0); i < members; i++ {
			member := fmt.Sprintf("%s-%d", cluster.Name, i)
			By(fmt.Sprintf("Checking that we can write to and read from %q", member))

			expected, err := framework.WriteSQLTest(cluster, member)
			Expect(err).NotTo(HaveOccurred())

			actual, err := framework.ReadSQLTest(cluster, member)
			Expect(err).NotTo(HaveOccurred())

			Expect(actual).To(Equal(expected))
		}
	})
})
