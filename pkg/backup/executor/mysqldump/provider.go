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
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"

	"github.com/golang/glog"

	utilexec "k8s.io/utils/exec"

	"github.com/oracle/mysql-operator/pkg/apis/mysql/v1"
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
func NewExecutor(executor *v1.Executor, creds map[string]string) (*Executor, error) {
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
	cmd := exec.Command(mysqldumpPath, append(args, ex.config.databases...)...)

	var mu sync.Mutex
	mu.Lock()
	defer mu.Unlock()

	tmpFile := path.Join(
		backupDir,
		fmt.Sprintf("%s.%s.sql.gz", clusterName, time.Now().UTC().Format("20060102150405")))

	f, err := os.Create(tmpFile)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()

	zw := gzip.NewWriter(f)
	defer zw.Close()
	cmd.SetStdout(zw)

	glog.V(4).Infof("running cmd: '%v'", cmd)
	err = cmd.Run()
	if err != nil {
		return nil, "", err
	}

	content, err := os.Open(tmpFile)
	if err != nil {
		return nil, "", err
	}
	return content, filepath.Base(tmpFile), nil
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

	glog.V(4).Infof("running cmd: '%v'", cmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		glog.V(4).Infof("err: '%v', output: '%s'", err, string(output))
		return err
	}
	return nil
}
