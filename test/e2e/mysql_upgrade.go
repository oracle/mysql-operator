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
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/oracle/mysql-operator/pkg/apis/mysql/v1alpha1"
	"github.com/oracle/mysql-operator/test/e2e/framework"
)

var _ = Describe("MySQL upgrade", func() {
	f := framework.NewDefaultFramework("mysql-update")

	It("should be possible to upgrade from MySQL 5.7.21 -> MySQL 5.7.22", func() {
		jig := framework.NewClusterTestJig(f.MySQLClientSet, f.ClientSet, "mysql-update")

		By("Creating a cluster with .spec.version = 5.7.21")
		cluster := jig.CreateAndAwaitClusterOrFail(f.Namespace.Name, 3, func(cluster *v1alpha1.Cluster) {
			cluster.Spec.Version = "5.7.21"
		}, framework.DefaultTimeout)

		expected, err := framework.WriteSQLTest(cluster, cluster.Name+"-0")
		Expect(err).NotTo(HaveOccurred())

		actual, err := framework.ReadSQLTest(cluster, cluster.Name+"-0")
		Expect(err).NotTo(HaveOccurred())
		Expect(actual).To(Equal(expected))

		By("Updating the cluster to .spec.version = 5.7.22")
		jig.UpdateClusterOrFail(cluster.Namespace, cluster.Name, func(cluster *v1alpha1.Cluster) {
			cluster.Spec.Version = "5.7.22"
		})

		jig.WaitForClusterReadyOrFail(cluster.Namespace, cluster.Name, framework.DefaultTimeout)

		By("Checking the Cluster's StatefulSet template has been updated to reflect the upgraded version")

		ss, err := jig.KubeClient.AppsV1beta1().StatefulSets(cluster.Namespace).Get(cluster.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		var found bool
		for _, container := range ss.Spec.Template.Spec.Containers {
			if container.Name == "mysql" && strings.Contains("5.7.22", container.Image) {
				found = true
			}
		}
		Expect(found).To(BeTrue(), "cluster StatefulSet template not updated to reflect upgraded version")
	})
})
