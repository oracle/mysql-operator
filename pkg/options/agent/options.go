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

package agent

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/pflag"
	glog "k8s.io/klog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// DefaultMySQLAgentHeathcheckPort is the port on which the mysql-agent's
	// healthcheck service runs on.
	DefaultMySQLAgentHeathcheckPort int32 = 10512
)

// MySQLAgentOpts holds the configuration options required to
// run the backup controller.
type MySQLAgentOpts struct {
	// HealthcheckPort is the port on which the mysql-agent healthcheck http
	// service runs on.
	HealthcheckPort int32

	// Address is the IP address to serve the backon. Set to 0.0.0.0 to listen
	// on all interfaces.
	Address string

	// Namespace is the namespace in which the backup controller (and is
	// associated Cluster) are running.
	Namespace string
	// ClusterName is the name of the Cluster the backup controller
	// is responsible for.
	ClusterName string
	// Hostname of the pod the backup operator is running in.
	Hostname string

	// minResyncPeriod is the resync period in reflectors; will be random
	// between minResyncPeriod and 2*minResyncPeriod.
	MinResyncPeriod metav1.Duration
}

// NewMySQLAgentOpts instantiates a new default
// MySQLAgentOpts getting values from the env where possible.
func NewMySQLAgentOpts() *MySQLAgentOpts {
	hostname, err := os.Hostname()
	if err != nil {
		glog.Fatalf("Failed to get the hostname: %v", err)
	}
	namespace := os.Getenv("POD_NAMESPACE")
	clusterName := os.Getenv("MYSQL_CLUSTER_NAME")
	return &MySQLAgentOpts{
		HealthcheckPort: DefaultMySQLAgentHeathcheckPort,
		Address:         "0.0.0.0",
		Namespace:       namespace,
		ClusterName:     clusterName,
		Hostname:        hostname,
		MinResyncPeriod: metav1.Duration{Duration: 12 * time.Hour},
	}
}

// AddFlags adds the mysql-agent flags to a given FlagSet.
func (s *MySQLAgentOpts) AddFlags(fs *pflag.FlagSet) *pflag.FlagSet {
	fs.Int32Var(&s.HealthcheckPort, "healthcheck-port", s.HealthcheckPort, "The port that the mysql-agent's healthcheck http service runs on.")
	fs.StringVar(&s.Address, "address", s.Address, "The IP address to serve the mysql-agent's http service on (set to 0.0.0.0 for all interfaces).")

	fs.StringVar(&s.Namespace, "namespace", s.Namespace, "The namespace to run in. Must be the same namespace as the associated MySQL cluster.")
	fs.StringVar(&s.ClusterName, "cluster-name", s.ClusterName, "The name of the MySQL cluster the mysql-agent is responsible for.")
	fs.StringVar(&s.Hostname, "hostname", s.Hostname, "The hostname of the pod the mysql-agent is running in.")
	fs.DurationVar(&s.MinResyncPeriod.Duration, "min-resync-period", s.MinResyncPeriod.Duration, "The resync period in reflectors will be random between MinResyncPeriod and 2*MinResyncPeriod.")

	return fs
}

// Validate checks that the required config options have been set.
func (s *MySQLAgentOpts) Validate() error {
	if len(s.Namespace) == 0 {
		return fmt.Errorf("must set --namespace or $POD_NAMESPACE")
	}
	if len(s.ClusterName) == 0 {
		return fmt.Errorf("must set --cluster-name or $MYSQL_CLUSTER_NAME")
	}
	if len(s.ClusterName) == 0 {
		return fmt.Errorf("failed to detect hostname. Set --hostname")
	}
	return nil
}
