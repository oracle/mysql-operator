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

package executor

import (
	"io"
	"os"

	"github.com/oracle/mysql-operator/pkg/apis/mysql/v1alpha1"
	"github.com/oracle/mysql-operator/pkg/backup/executor/mysqldump"
)

const (
	// MySQLDumpProvider denotes the mysqldump utility backup and restore provider.
	MySQLDumpProvider = "mysqldump"
)

// ExecutorProviders denotes the list of available ExecutorProviders.
var ExecutorProviders = [...]string{MySQLDumpProvider}

// Interface will execute backup operations via a tool such as mysqlbackup or
// mysqldump.
type Interface interface {
	// Backup runs a backup operation using the given credentials, returning the content.
	// TODO: default backupDir to allow streaming...
	Backup(backupDir string, clusterName string) (io.ReadCloser, string, error)
	// Restore restores the given content to the mysql node.
	Restore(content io.ReadCloser) error
}

// New builds a new backup executor.
func New(executor v1alpha1.BackupExecutor, creds map[string]string) (Interface, error) {
	return mysqldump.NewExecutor(executor.MySQLDump, creds)
}

// DefaultCreds return the default MySQL credentials for the local instance.
func DefaultCreds() map[string]string {
	return map[string]string{
		"username": "root",
		"password": os.Getenv("MYSQL_ROOT_PASSWORD"),
	}
}
