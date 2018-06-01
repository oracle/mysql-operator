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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"

	"github.com/oracle/mysql-operator/pkg/apis/mysql/v1alpha1"
	mysqlclientset "github.com/oracle/mysql-operator/pkg/generated/clientset/versioned"
	"github.com/oracle/mysql-operator/test/e2e/framework"
)

var _ = Describe("Pod crash", func() {
	f := framework.NewDefaultFramework("pod-crash")

	var cs clientset.Interface
	var mcs mysqlclientset.Interface
	BeforeEach(func() {
		cs = f.ClientSet
		mcs = f.MySQLClientSet
	})

	It("should be the case that single-primary Clusters recover from Pods crashing", func() {
		clusterName := "pod-crash"
		ns := f.Namespace.Name
		grace := int64(0) // kill don't gracefully terminate

		jig := framework.NewClusterTestJig(mcs, cs, clusterName)

		cluster := jig.CreateAndAwaitClusterOrFail(ns, 3, nil, framework.DefaultTimeout)

		primary := framework.GetReadyPrimaryPodName(cs, ns, cluster.Name)

		expected, err := framework.WriteSQLTest(cluster, primary)
		Expect(err).NotTo(HaveOccurred())

		actual, err := framework.ReadSQLTest(cluster, primary)
		Expect(err).NotTo(HaveOccurred())
		Expect(actual).To(Equal(expected))

		By("Terminating the cluster primary")

		err = cs.CoreV1().Pods(ns).Delete(primary, &metav1.DeleteOptions{GracePeriodSeconds: &grace})
		Expect(err).NotTo(HaveOccurred())

		By("Checking that the primary has failed over to another member")

		newPrimary := framework.GetReadyPrimaryPodName(cs, ns, cluster.Name)
		Expect(newPrimary).NotTo(Equal(primary))

		secondary := framework.GetReadySecondaryPodName(cs, ns, cluster.Name)

		By(fmt.Sprintf("Terminating cluster secondary %q", secondary))

		err = cs.CoreV1().Pods(ns).Delete(secondary, &metav1.DeleteOptions{GracePeriodSeconds: &grace})
		Expect(err).NotTo(HaveOccurred())

		By("Checking that we can still read the previously inserted data from the test DB")
		actual, err = framework.ReadSQLTest(cluster, newPrimary)
		Expect(err).NotTo(HaveOccurred())
		Expect(actual).To(Equal(expected))
	})

	It("should be the case that multi-primary Clusters recover from Pods crashing", func() {
		clusterName := "pod-crash"
		ns := f.Namespace.Name
		grace := int64(0) // kill don't gracefully terminate

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

		By(fmt.Sprintf("Terminating cluster member %q", primary))

		err = cs.CoreV1().Pods(ns).Delete(primary, &metav1.DeleteOptions{GracePeriodSeconds: &grace})
		Expect(err).NotTo(HaveOccurred())

		By("Checking that we can still read the previously inserted data from the test DB")

		actual, err = framework.ReadSQLTest(cluster, framework.GetReadyPrimaryPodName(cs, ns, cluster.Name))
		Expect(err).NotTo(HaveOccurred())
		Expect(actual).To(Equal(expected))
	})
})
