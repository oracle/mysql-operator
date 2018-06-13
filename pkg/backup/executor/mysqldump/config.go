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

package mysqldump

import (
	"fmt"

	"github.com/oracle/mysql-operator/pkg/apis/mysql/v1alpha1"
)

// Config holds the MySQL credentials required to authenticate with the MySQL database being
// backed-up or restored.
type Config struct {
	username  string
	password  string
	databases []v1alpha1.Database
}

// NewConfig creates an mysqldump configuration based on the input parameters.
func NewConfig(executor *v1alpha1.MySQLDumpBackupExecutor, creds map[string]string) *Config {
	return &Config{
		databases: executor.Databases,
		username:  creds["username"],
		password:  creds["password"],
	}
}

// Validate checks the required configuration parameters are set.
func (c Config) Validate() (err error) {
	if c.username == "" {
		return fmt.Errorf("no mysqldump 'username' provided")
	}
	if c.password == "" {
		return fmt.Errorf("no mysqldump 'password' provided")
	}
	if c.databases == nil || len(c.databases) == 0 {
		return fmt.Errorf("no mysqldump 'databases' provided")
	}
	return nil
}
