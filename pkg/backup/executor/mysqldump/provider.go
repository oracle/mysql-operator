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
	"compress/gzip"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"

	utilexec "k8s.io/utils/exec"

	"github.com/oracle/mysql-operator/pkg/apis/mysql/v1alpha1"
)

const (
	mysqldumpCmd = "mysqldump"
	mysqlCmd     = "mysql"
)

// Executor creates backups using mysqldump.
type Executor struct {
	config *Config
}

// NewExecutor creates a provider capable of creating and restoring backups with the mysqldump
// tool.
func NewExecutor(executor *v1alpha1.MySQLDumpBackupExecutor, creds map[string]string) (*Executor, error) {
	cfg := NewConfig(executor, creds)
	err := cfg.Validate()
	if err != nil {
		return nil, err
	}
	return &Executor{config: cfg}, nil
}

// Backup performs a full cluster backup using the mysqldump tool.
func (ex *Executor) Backup(backupDir string, clusterName string) (io.ReadCloser, string, error) {
	exec := utilexec.New()
	mysqldumpPath, err := exec.LookPath(mysqldumpCmd)
	if err != nil {
		return nil, "", fmt.Errorf("mysqldump path: %v", err)
	}

	args := []string{
		"-u" + ex.config.username,
		"-p" + ex.config.password,
		"--single-transaction",
		"--skip-lock-tables",
		"--flush-privileges",
		"--set-gtid-purged=OFF",
		"--databases",
	}

	dbNames := make([]string, len(ex.config.databases))
	for i, database := range ex.config.databases {
		dbNames[i] = database.Name
	}

	cmd := exec.Command(mysqldumpPath, append(args, dbNames...)...)

	var mu sync.Mutex
	mu.Lock()
	defer mu.Unlock()

	backupName := fmt.Sprintf("%s.%s.sql.gz", clusterName, time.Now().UTC().Format("20060102150405"))

	pr, pw := io.Pipe()
	zw := gzip.NewWriter(pw)
	cmd.SetStdout(zw)

	go func() {
		glog.V(4).Infof("running cmd: '%s %s'", mysqldumpPath, SanitizeArgs(append(args, dbNames...), ex.config.password))
		err = cmd.Run()
		zw.Close()
		if err != nil {
			pw.CloseWithError(errors.Wrap(err, "executing backup"))
		} else {
			pw.Close()
		}
	}()

	return pr, backupName, nil
}

// Restore a cluster from a mysqldump.
func (ex *Executor) Restore(content io.ReadCloser) error {
	defer content.Close()

	exec := utilexec.New()
	mysqlPath, err := exec.LookPath(mysqlCmd)
	if err != nil {
		return fmt.Errorf("mysql path: %v", err)
	}

	args := []string{
		"-u" + ex.config.username,
		"-p" + ex.config.password,
	}
	cmd := exec.Command(mysqlPath, args...)

	var mu sync.Mutex
	mu.Lock()
	defer mu.Unlock()

	zr, err := gzip.NewReader(content)
	if err != nil {
		return err
	}
	defer zr.Close()
	cmd.SetStdin(zr)

	glog.V(4).Infof("running cmd: '%s %s'", mysqlPath, SanitizeArgs(args, ex.config.password))
	_, err = cmd.CombinedOutput()
	return err
}

// SanitizeArgs takes a slice, redacts all occurrences of a given string and
// returns a single string concatenated with spaces
func SanitizeArgs(args []string, old string) string {
	return strings.Replace(strings.Join(args, " "), old, "REDACTED", -1)
}
