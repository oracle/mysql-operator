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
	"context"
	"fmt"
	"reflect"
	"testing"

	"k8s.io/utils/exec"
	fakeexec "k8s.io/utils/exec/testing"
)

const clusterStatusOutput = `
mysqlx: [Warning] Using a password on the command line interface can be insecure.
{
    "clusterName": "MySQLCluster",
    "defaultReplicaSet": {
        "name": "default",
        "primary": "mysql-test-cluster-0.service.namespace.svc.cluster.local:3306",
        "status": "OK",
        "statusText": "Cluster is ONLINE and can tolerate up to ONE failure.",
        "topology": {
            "mysql-test-cluster-0.service.namespace.svc.cluster.local:3306": {
                "address": "mysql-test-cluster-0.service.namespace.svc.cluster.local:3306",
                "mode": "R/O",
                "readReplicas": {},
                "role": "HA",
                "status": "MISSING"
            },
            "mysql-test-cluster-1.service.namespace.svc.cluster.local:3306": {
                "address": "mysql-test-cluster-1.service.namespace.svc.cluster.local:3306",
                "mode": "R/O",
                "readReplicas": {},
                "role": "HA",
                "status": "ONLINE"
            },
            "mysql-test-cluster-2.service.namespace.svc.cluster.local:3306": {
                "address": "mysql-test-cluster-2.service.namespace.svc.cluster.local:3306",
                "mode": "R/W",
                "readReplicas": {},
                "role": "HA",
                "status": "ONLINE"
            }
        }
    }
}`

func TestStripPasswordWarning(t *testing.T) {
	input := []byte(`mysqlx: [Warning] Using a password on the command line interface can be insecure.
{
    "hello" : "there"
}`)
	expected := []byte(`{
    "hello" : "there"
}`)
	output := (&runner{}).stripPasswordWarning(input)
	if !reflect.DeepEqual(output, expected) {
		t.Errorf("runner.stripPasswordWarning(%q), => (%q), expected (%q)", input, output, expected)
	}
}

func TestGetClusterStatus(t *testing.T) {
	warning := "No entry for terminal type \"unknown\";\nusing dumb terminal settings.\n"

	fcmd := fakeexec.FakeCmd{
		RunScript: []fakeexec.FakeRunAction{
			func() ([]byte, []byte, error) { return []byte(clusterStatusOutput), []byte(warning), nil },
		},
	}
	fexec := fakeexec.FakeExec{
		CommandScript: []fakeexec.FakeCommandAction{
			func(cmd string, args ...string) exec.Cmd { return fakeexec.InitFakeCmd(&fcmd, cmd, args...) },
		},
	}

	uri := "root:foo@localhost.service.namespace.svc.cluster.local:3306"
	runner := New(&fexec, uri)
	ctx := context.Background()
	status, err := runner.GetClusterStatus(ctx)

	if fcmd.RunCalls != 1 {
		t.Errorf("Expected 1 exec('mysqlsh'), got %d", fcmd.RunCalls)
	}

	expectedCall := []string{
		"mysqlsh",
		"--uri", "root:foo@localhost.service.namespace.svc.cluster.local:3306",
		"--py",
		"-e", "print dba.get_cluster('MySQLCluster').status()",
	}
	if !reflect.DeepEqual(fcmd.RunLog[0], expectedCall) {
		t.Errorf("Expected call %+v, got %+v", expectedCall, fcmd.RunLog[0])
	}

	if err != nil {
		t.Fatalf("Expected err = nil, got: %v", err)
	}

	if status.ClusterName != "MySQLCluster" {
		t.Errorf("Expected status.ClusterName = \"MySQLCluster\", got %q", status.ClusterName)
	}

	n := len(status.DefaultReplicaSet.Topology)
	if n != 3 {
		t.Errorf("Expected 3 instances in status.DefaultReplicaSet.Topology, got %d", n)
	}
}

func TestGetInstanceStatus(t *testing.T) {
	getInstanceStateOutput := `
mysqlx: [Warning] Using a password on the command line interface can be insecure.
{"reason": "recoverable", "state": "ok"}`
	warning := "No entry for terminal type \"unknown\";\nusing dumb terminal settings.\n"
	fcmd := fakeexec.FakeCmd{
		RunScript: []fakeexec.FakeRunAction{
			func() ([]byte, []byte, error) { return []byte(getInstanceStateOutput), []byte(warning), nil },
		},
	}
	fexec := fakeexec.FakeExec{
		CommandScript: []fakeexec.FakeCommandAction{
			func(cmd string, args ...string) exec.Cmd { return fakeexec.InitFakeCmd(&fcmd, cmd, args...) },
		},
	}

	uri := "root:foo@localhost.service.namespace.svc.cluster.local:3306"
	instanceURI := "root:foo@mysql-test-cluster-2.service.namespace.svc.cluster.local:3306"
	runner := New(&fexec, uri)
	ctx := context.Background()
	state, err := runner.CheckInstanceState(ctx, instanceURI)

	if fcmd.RunCalls != 1 {
		t.Errorf("Expected 1 exec('mysqlsh'), got %d", fcmd.RunCalls)
	}

	expectedCall := []string{
		"mysqlsh",
		"--uri", "root:foo@localhost.service.namespace.svc.cluster.local:3306",
		"--py",
		"-e", fmt.Sprintf("print dba.get_cluster('MySQLCluster').check_instance_state('%s')", instanceURI),
	}
	if !reflect.DeepEqual(fcmd.RunLog[0], expectedCall) {
		t.Errorf("Expected call %+v, got %+v", expectedCall, fcmd.RunLog[0])
	}

	if err != nil {
		t.Fatalf("Expected err = nil, got: %v", err)
	}

	if state.State != "ok" {
		t.Errorf("Expected state.State = \"ok\", got %q", state.State)
	}
	if state.Reason != "recoverable" {
		t.Errorf("Expected state.Reason = \"recoverable\", got %q", state.Reason)
	}
}

func TestRemoveInstanceFromCluster(t *testing.T) {
	fcmd := fakeexec.FakeCmd{
		RunScript: []fakeexec.FakeRunAction{
			func() ([]byte, []byte, error) { return []byte{}, []byte{}, nil },
		},
	}
	fexec := fakeexec.FakeExec{
		CommandScript: []fakeexec.FakeCommandAction{
			func(cmd string, args ...string) exec.Cmd { return fakeexec.InitFakeCmd(&fcmd, cmd, args...) },
		},
	}

	uri := "root:foo@localhost:3306"
	runner := New(&fexec, uri)
	ctx := context.Background()
	err := runner.RemoveInstanceFromCluster(ctx, "root:foo@mysql-cluster-1:3306")

	if fcmd.RunCalls != 1 {
		t.Errorf("Expected 1 exec('mysqlsh'), got %d", fcmd.RunCalls)
	}

	expectedCall := []string{
		"mysqlsh",
		"--uri", "root:foo@localhost:3306",
		"--py",
		"-e", "dba.get_cluster('MySQLCluster').remove_instance('root:foo@mysql-cluster-1:3306', {\"force\":True})",
	}
	if !reflect.DeepEqual(fcmd.RunLog[0], expectedCall) {
		t.Errorf("Expected call %+v, got %+v", expectedCall, fcmd.RunLog[0])
	}

	if err != nil {
		t.Fatalf("Expected err = nil, got: %v", err)
	}
}
