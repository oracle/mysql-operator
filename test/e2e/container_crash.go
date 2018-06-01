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

	clientset "k8s.io/client-go/kubernetes"

	"github.com/oracle/mysql-operator/pkg/apis/mysql/v1alpha1"
	mysqlclientset "github.com/oracle/mysql-operator/pkg/generated/clientset/versioned"
	"github.com/oracle/mysql-operator/test/e2e/framework"
)

var _ = Describe("Container crash", func() {
	f := framework.NewDefaultFramework("container-crash")

	var cs clientset.Interface
	var mcs mysqlclientset.Interface
	BeforeEach(func() {
		cs = f.ClientSet
		mcs = f.MySQLClientSet
	})

	It("should be the case that single-primary Clusters recover from mysql-server containers crashing", func() {
		clusterName := "mysql-server-crash"
		ns := f.Namespace.Name

		jig := framework.NewClusterTestJig(mcs, cs, clusterName)

		cluster := jig.CreateAndAwaitClusterOrFail(ns, 3, nil, framework.DefaultTimeout)

		primary := framework.GetReadyPrimaryPodName(cs, ns, cluster.Name)

		expected, err := framework.WriteSQLTest(cluster, primary)
		Expect(err).NotTo(HaveOccurred())

		actual, err := framework.ReadSQLTest(cluster, primary)
		Expect(err).NotTo(HaveOccurred())
		Expect(actual).To(Equal(expected))

		By("Terminating the primary mysql-server process")

		_, err = framework.RunHostCmd(ns, primary, "mysql", "kill -9 1")
		Expect(err).NotTo(HaveOccurred())

		By("Checking the Pod and MySQL cluster both become ready again")

		framework.AwaitPodReadyOrDie(cs, ns, primary, framework.DefaultTimeout)
		jig.WaitForClusterReadyOrFail(ns, cluster.Name, framework.DefaultTimeout)

		secondary := framework.GetReadySecondaryPodName(cs, ns, cluster.Name)

		By(fmt.Sprintf("Terminating mysql-server process of cluster secondary %q", secondary))

		_, err = framework.RunHostCmd(ns, secondary, "mysql", "kill -9 1")
		Expect(err).NotTo(HaveOccurred())

		By("Checking the Pod and MySQL cluster both become ready again")

		framework.AwaitPodReadyOrDie(cs, ns, primary, framework.DefaultTimeout)
		jig.WaitForClusterReadyOrFail(ns, cluster.Name, framework.DefaultTimeout)

		By("Checking that we can still read the previously inserted data from the test DB")
		actual, err = framework.ReadSQLTest(cluster, primary)
		Expect(err).NotTo(HaveOccurred())
		Expect(actual).To(Equal(expected))
	})

	It("should be the case that multi-primary Clusters recover from mysql-server containers crashing", func() {
		clusterName := "mysql-server-crash"
		ns := f.Namespace.Name

		jig := framework.NewClusterTestJig(mcs, cs, clusterName)

		cluster := jig.CreateAndAwaitClusterOrFail(ns, 3, func(cluster *v1alpha1.Cluster) {
			cluster.Spec.MultiMaster = true
		}, framework.DefaultTimeout)

		primary := framework.GetReadyPrimaryPodName(cs, ns, cluster.Name)

		expected, err := framework.WriteSQLTest(cluster, primary)
		Expect(err).NotTo(HaveOccurred())

		actual, err := framework.ReadSQLTest(cluster, primary)
		Expect(err).NotTo(HaveOccurred())
		Expect(actual).To(Equal(expected))

		By("Terminating the primary mysql-server process")

		_, err = framework.RunHostCmd(ns, primary, "mysql", "kill -9 1")
		Expect(err).NotTo(HaveOccurred())

		By("Checking the Pod and MySQL cluster both become ready again")

		framework.AwaitPodReadyOrDie(cs, ns, primary, framework.DefaultTimeout)
		jig.WaitForClusterReadyOrFail(ns, cluster.Name, framework.DefaultTimeout)

		By("Checking that we can still read the previously inserted data from the test DB")
		actual, err = framework.ReadSQLTest(cluster, primary)
		Expect(err).NotTo(HaveOccurred())
		Expect(actual).To(Equal(expected))
	})

	It("should be the case that single-primary Clusters recover from mysql-agent containers crashing", func() {
		clusterName := "mysql-agent-crash"
		ns := f.Namespace.Name

		jig := framework.NewClusterTestJig(mcs, cs, clusterName)

		cluster := jig.CreateAndAwaitClusterOrFail(ns, 3, nil, framework.DefaultTimeout)

		primary := framework.GetReadyPrimaryPodName(cs, ns, cluster.Name)

		expected, err := framework.WriteSQLTest(cluster, primary)
		Expect(err).NotTo(HaveOccurred())

		actual, err := framework.ReadSQLTest(cluster, primary)
		Expect(err).NotTo(HaveOccurred())
		Expect(actual).To(Equal(expected))

		By("Terminating the primary mysql-agent process")

		_, err = framework.RunHostCmd(ns, primary, "mysql-agent", "kill -9 1")
		Expect(err).NotTo(HaveOccurred())

		By("Checking the Pod and MySQL cluster both become ready again")

		framework.AwaitPodReadyOrDie(cs, ns, primary, framework.DefaultTimeout)
		jig.WaitForClusterReadyOrFail(ns, cluster.Name, framework.DefaultTimeout)

		secondary := framework.GetReadySecondaryPodName(cs, ns, cluster.Name)

		By(fmt.Sprintf("Terminating mysql-agent process of cluster secondary %q", secondary))

		_, err = framework.RunHostCmd(ns, secondary, "mysql-agent", "kill -9 1")
		Expect(err).NotTo(HaveOccurred())

		By("Checking the Pod and MySQL cluster both become ready again")

		framework.AwaitPodReadyOrDie(cs, ns, primary, framework.DefaultTimeout)
		jig.WaitForClusterReadyOrFail(ns, cluster.Name, framework.DefaultTimeout)

		By("Checking that we can still read the previously inserted data from the test DB")
		actual, err = framework.ReadSQLTest(cluster, primary)
		Expect(err).NotTo(HaveOccurred())
		Expect(actual).To(Equal(expected))
	})

	It("should be the case that multi-primary Clusters recover from mysql-agent containers crashing", func() {
		clusterName := "mysql-agent-crash"
		ns := f.Namespace.Name

		jig := framework.NewClusterTestJig(mcs, cs, clusterName)

		cluster := jig.CreateAndAwaitClusterOrFail(ns, 3, func(cluster *v1alpha1.Cluster) {
			cluster.Spec.MultiMaster = true
		}, framework.DefaultTimeout)

		primary := framework.GetReadyPrimaryPodName(cs, ns, cluster.Name)

		expected, err := framework.WriteSQLTest(cluster, primary)
		Expect(err).NotTo(HaveOccurred())

		actual, err := framework.ReadSQLTest(cluster, primary)
		Expect(err).NotTo(HaveOccurred())
		Expect(actual).To(Equal(expected))

		By("Terminating the primary mysql-agent process")

		_, err = framework.RunHostCmd(ns, primary, "mysql-agent", "kill -9 1")
		Expect(err).NotTo(HaveOccurred())

		By("Checking the Pod and MySQL cluster both become ready again")

		framework.AwaitPodReadyOrDie(cs, ns, primary, framework.DefaultTimeout)
		jig.WaitForClusterReadyOrFail(ns, cluster.Name, framework.DefaultTimeout)

		By("Checking that we can still read the previously inserted data from the test DB")
		actual, err = framework.ReadSQLTest(cluster, primary)
		Expect(err).NotTo(HaveOccurred())
		Expect(actual).To(Equal(expected))
	})
})
