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

	corev1 "k8s.io/api/core/v1"

	"github.com/oracle/mysql-operator/pkg/apis/mysql/v1alpha1"
	"github.com/oracle/mysql-operator/test/e2e/framework"

	mysqlclientset "github.com/oracle/mysql-operator/pkg/generated/clientset/versioned"
)

var _ = Describe("Backup/Restore", func() {
	f := framework.NewDefaultFramework("backup-restore")

	var cs clientset.Interface
	var mcs mysqlclientset.Interface
	BeforeEach(func() {
		cs = f.ClientSet
		mcs = f.MySQLClientSet
	})

	It("should be possible to backup a cluster and restore the created backup", func() {
		clusterName := "backup-restore"
		ns := f.Namespace.Name

		clusterJig := framework.NewClusterTestJig(mcs, cs, clusterName)
		backupJig := framework.NewBackupTestJig(mcs, cs, clusterName)
		restoreJig := framework.NewRestoreTestJig(mcs, cs, clusterName)

		By("Creating a cluster to backup")

		cluster := clusterJig.CreateAndAwaitClusterOrFail(ns, 3, nil, framework.DefaultTimeout)

		By("Creating testdb in the cluster to be backed up")

		member := cluster.Name + "-0"
		expected, err := framework.WriteSQLTest(cluster, member)
		Expect(err).NotTo(HaveOccurred())

		By("Checking testdb is present")

		actual, err := framework.ReadSQLTest(cluster, member)
		Expect(err).NotTo(HaveOccurred())
		Expect(actual).To(Equal(expected))

		By("creating a secret containing the S3 (compat.) upload credentials")

		secret, err := backupJig.CreateS3AuthSecret(ns, "s3-upload-creds")
		Expect(err).NotTo(HaveOccurred())

		By("Backing up testdb")

		dbs := []v1alpha1.Database{{Name: framework.TestDBName}}
		backup := backupJig.CreateAndAwaitMySQLDumpBackupOrFail(ns, clusterName, dbs, func(b *v1alpha1.Backup) {
			b.Spec.StorageProvider = v1alpha1.StorageProvider{
				S3: &v1alpha1.S3StorageProvider{
					Endpoint:       "bristoldev.compat.objectstorage.us-phoenix-1.oraclecloud.com",
					Region:         "us-phoenix-1",
					Bucket:         "trjl-test",
					ForcePathStyle: true,
					CredentialsSecret: &corev1.LocalObjectReference{
						Name: secret.Name,
					},
				},
			}
		}, framework.DefaultTimeout)

		Expect(backup.Status.Outcome.Location).NotTo(BeEmpty())

		By("Dropping testdb")

		_, err = framework.ExecuteSQL(cluster, member,
			fmt.Sprintf("DROP DATABASE IF EXISTS %s", framework.TestDBName))
		Expect(err).NotTo(HaveOccurred())

		By("Checking that testdb has been dropped")

		_, err = framework.ReadSQLTest(cluster, member)
		Expect(err).To(HaveOccurred())

		By("Restoring the backup")

		restore := restoreJig.CreateAndAwaitRestoreOrFail(ns, clusterName, backup.Name, nil, framework.DefaultTimeout)
		Expect(restore.Status.TimeCompleted).ToNot(BeZero())

		By("Checking testdb is present and contains the correct uuid")

		actual, err = framework.ReadSQLTest(cluster, member)
		Expect(err).NotTo(HaveOccurred())
		Expect(actual).To(Equal(expected))
	})
})
