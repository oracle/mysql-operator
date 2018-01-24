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

package options

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestEnsureDefaults(t *testing.T) {
	server := MySQLOperatorServer{}
	server.EnsureDefaults()
	assertRequiredDefaults(t, server)
}

func assertRequiredDefaults(t *testing.T, s MySQLOperatorServer) {
	if &s == nil {
		t.Error("MySQLOperatorServer: was nil, expected a valid configuration.")
	}
	if len(s.Hostname) <= 0 {
		t.Errorf("MySQLOperatorServer: expected a non-zero length hostname, got '%s'.", s.Hostname)
	}
	if &s.Images == nil {
		t.Error("MySQLOperatorServer.Images: was nil, expected a valid configuration.")
	}
	if s.Images.MySQLServerImage != mysqlServer {
		t.Errorf("MySQLOperatorServer.Images.MySQLServerImage: was '%s', expected '%s'.", s.Images.MySQLServerImage, mysqlServer)
	}
	if s.Images.MySQLAgentImage != mysqlAgent {
		t.Errorf("MySQLOperatorServer.Images.MySQLAgentImage: was '%s', expected '%s'.", s.Images.MySQLAgentImage, mysqlAgent)
	}
	expectedDuration := v1.Duration{Duration: 43200000000000}
	if &s.MinResyncPeriod == nil {
		t.Errorf("MySQLOperatorServer.MinResyncPeriod: was nil, expected '%s'.", expectedDuration)
	}
	if s.MinResyncPeriod != expectedDuration {
		t.Errorf("MySQLOperatorServer.MinResyncPeriod: was '%s', expected '%s'.", s.MinResyncPeriod, expectedDuration)
	}
}

func TestEnsureDefaultsOverrideSafety(t *testing.T) {
	expected := mockMySQLOperatorServer()
	ensured := mockMySQLOperatorServer()
	ensured.EnsureDefaults()
	if expected != ensured {
		t.Errorf("MySQLOperatorServer.EnsureDefaults() should not modify pre-configured values.")
	}
}

func mockMySQLOperatorServer() MySQLOperatorServer {
	return MySQLOperatorServer{
		KubeConfig: "some-kube-config",
		Master:     "some-master",
		Hostname:   "some-hostname",
		Images: Images{
			MySQLServerImage: "some-mysql-img",
			MySQLAgentImage:  "some-agent-img",
		},
		MinResyncPeriod: v1.Duration{Duration: 42},
	}
}
