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

package mysqlsh

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/golang/glog"
	"github.com/pkg/errors"

	utilexec "k8s.io/utils/exec"

	"github.com/oracle/mysql-operator/pkg/cluster/innodb"
)

// Interface is an injectable interface for running mysqlsh commands.
type Interface interface {
	IsClustered(ctx context.Context) bool
	// CreateCluster creates a new InnoDB cluster called
	// innodb.DefaultClusterName.
	CreateCluster(ctx context.Context, multiMaster bool) (*innodb.ClusterStatus, error)
	// GetClusterStatus gets the status of the innodb.DefaultClusterName InnoDB
	// cluster.
	GetClusterStatus(ctx context.Context) (*innodb.ClusterStatus, error)
	// CheckInstanceState verifies the existing data on the instance (specified
	// by URI) does not prevent it from joining a cluster.
	CheckInstanceState(ctx context.Context, uri string) (*innodb.InstanceState, error)
	// AddInstanceToCluster adds the instance (specified by URI) the InnoDB
	// cluster.
	AddInstanceToCluster(ctx context.Context, uri string) error
	// RejoinInstanceToCluster rejoins an instance (specified by URI) to the
	// InnoDB cluster.
	RejoinInstanceToCluster(ctx context.Context, uri string) error
	// RemoveInstanceFromCluster removes an instance (specified by URI) to the
	// InnoDB cluster.
	RemoveInstanceFromCluster(ctx context.Context, uri string) error
}

// New creates a new MySQL Shell Interface.
func New(exec utilexec.Interface, uri string) Interface {
	return &runner{exec: exec, uri: uri}
}

// runner implements Interface in terms of exec("mysqlsh").
type runner struct {
	mu   sync.Mutex
	exec utilexec.Interface

	// uri is Uniform Resource Identifier of the MySQL instance to connect to.
	// Format: [user[:pass]]@host[:port][/db].
	uri string
}

func (r *runner) IsClustered(ctx context.Context) bool {
	python := fmt.Sprintf("dba.get_cluster('%s')", innodb.DefaultClusterName)
	_, err := r.run(ctx, python)
	return err == nil
}

func (r *runner) CreateCluster(ctx context.Context, multiMaster bool) (*innodb.ClusterStatus, error) {
	var python string
	if multiMaster {
		python = fmt.Sprintf("dba.create_cluster('%s', {'force':True,'multiMaster':True})", innodb.DefaultClusterName)
	} else {
		python = fmt.Sprintf("dba.create_cluster('%s')", innodb.DefaultClusterName)
	}
	_, err := r.run(ctx, python)
	if err != nil {
		return nil, fmt.Errorf("creating cluster: %v", err)
	}
	return r.GetClusterStatus(ctx)
}

func (r *runner) GetClusterStatus(ctx context.Context) (*innodb.ClusterStatus, error) {
	python := fmt.Sprintf("print dba.get_cluster('%s').status()", innodb.DefaultClusterName)
	output, err := r.run(ctx, python)
	if err != nil {
		return nil, errors.Wrap(err, "GetClusterStatus failed")
	}

	status := &innodb.ClusterStatus{}
	err = json.Unmarshal(output, status)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("decoding cluster status output: '%s'", output))
	}

	return status, nil
}

func (r *runner) CheckInstanceState(ctx context.Context, uri string) (*innodb.InstanceState, error) {
	python := fmt.Sprintf("print dba.get_cluster('%s').check_instance_state('%s')", innodb.DefaultClusterName, uri)
	output, err := r.run(ctx, python)

	if err != nil {
		return nil, err
	}

	state := &innodb.InstanceState{}
	err = json.Unmarshal(output, state)
	if err != nil {
		return nil, fmt.Errorf("decoding instance state: %v", err)
	}

	return state, nil
}

func (r *runner) AddInstanceToCluster(ctx context.Context, uri string) error {
	python := fmt.Sprintf("dba.get_cluster('%s').add_instance('%s')", innodb.DefaultClusterName, uri)
	_, err := r.run(ctx, python)
	return err
}

func (r *runner) RejoinInstanceToCluster(ctx context.Context, uri string) error {
	python := fmt.Sprintf("dba.get_cluster('%s').rejoin_instance('%s')", innodb.DefaultClusterName, uri)
	_, err := r.run(ctx, python)
	return err
}

func (r *runner) RemoveInstanceFromCluster(ctx context.Context, uri string) error {
	python := fmt.Sprintf("dba.get_cluster('%s').remove_instance('%s', {\"force\":True})", innodb.DefaultClusterName, uri)
	_, err := r.run(ctx, python)
	return err
}

// stripPasswordWarning strips the password warning output by mysqlsh due to the
// fact we pass the password as part of the connection URI.
func (r *runner) stripPasswordWarning(in []byte) []byte {
	warning := []byte("mysqlx: [Warning] Using a password on the command line interface can be insecure.\n")
	return bytes.Replace(in, warning, []byte(""), 1)
}

func (r *runner) run(ctx context.Context, python string) ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	args := []string{"--uri", r.uri, "--py", "-e", python}

	cmd := r.exec.CommandContext(ctx, "mysqlsh", args...)

	cmd.SetStdout(stdout)
	cmd.SetStderr(stderr)

	glog.V(6).Infof("Running command: mysqlsh %v", args)
	err := cmd.Run()
	glog.V(6).Infof("    stdout: %s\n    stderr: %s\n    err: %s", stdout, stderr, err)

	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("mysqlsh %s: err=%+v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), err, stdout, stderr))
	}
	return r.stripPasswordWarning(stdout.Bytes()), err
}
