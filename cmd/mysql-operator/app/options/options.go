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
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v2"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	mysqlServer = "mysql/mysql-server"
	mysqlAgent  = "iad.ocir.io/oracle/mysql-agent"
)

// Images is the configuration of required MySQLOperator images. Remember to configure the appropriate
// credentials for the target repositories.
type Images struct {
	MySQLServerImage string `yaml:"mysqlServer"`
	MySQLAgentImage  string `yaml:"mysqlAgent"`
}

// MySQLOperatorServer holds the options for the MySQLOperator.
type MySQLOperatorServer struct {
	// KubeConfig is the path to a kubeconfig file, specifying how to connect to
	// the API server.
	KubeConfig string `yaml:"kubeconfig"`

	// Master is the address of the Kubernetes API server (overrides any value
	// in kubeconfig).
	Master string `yaml:"master"`

	// Namespace is the (optional) namespace in which the MySQL operator will
	// manage MySQL Clusters. Defaults to metav1.NamespaceAll.
	Namespace string `yaml:"namespace"`

	// Hostname of the pod the operator is running in.
	Hostname string `yaml:"hostname"`

	// Images defines the 'mysql-server' and 'mysql-agent' images to use.
	Images Images `yaml:"images"`

	// minResyncPeriod is the resync period in reflectors; will be random
	// between minResyncPeriod and 2*minResyncPeriod.
	MinResyncPeriod metav1.Duration `yaml:"minResyncPeriod"`
}

// NewMySQLOperatorServer will create a new MySQLOperatorServer. If a valid
// config file is specified and exists, it will be used to initialise the
// server. Otherwise, a default server will be created.
//
// The values specified by either default may later be customised and overidden
// by user specified commandline parameters.
func NewMySQLOperatorServer(filePath string) (*MySQLOperatorServer, error) {
	var config MySQLOperatorServer
	yamlPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to determine MySQLOperator configuration absolute path: '%s'", filePath)
	}
	if _, err := os.Stat(filePath); err == nil {
		yamlFile, err := ioutil.ReadFile(yamlPath)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read MySQLOperator configuration: '%s'", filePath)
		}
		err = yaml.Unmarshal(yamlFile, &config)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse MySQLOperator configuration: '%s'", filePath)
		}
	} else {
		config = MySQLOperatorServer{}
	}
	config.EnsureDefaults()
	return &config, nil
}

// EnsureDefaults provides a default configuration when required values have
// not been set.
func (s *MySQLOperatorServer) EnsureDefaults() {
	if s.Hostname == "" {
		hostname, err := os.Hostname()
		if err != nil {
			glog.Fatalf("Failed to get the hostname: %v", err)
		}
		s.Hostname = hostname
	}
	if &s.Images == nil {
		s.Images = Images{}
	}
	if s.Images.MySQLServerImage == "" {
		s.Images.MySQLServerImage = mysqlServer
	}
	if s.Images.MySQLAgentImage == "" {
		s.Images.MySQLAgentImage = mysqlAgent
	}
	if s.MinResyncPeriod.Duration <= 0 {
		s.MinResyncPeriod = metav1.Duration{Duration: 12 * time.Hour}
	}
}

// AddFlags adds the mysql-operator flags to a given FlagSet.
func (s *MySQLOperatorServer) AddFlags(fs *pflag.FlagSet) *pflag.FlagSet {
	fs.StringVar(&s.KubeConfig, "kubeconfig", s.KubeConfig, "Path to Kubeconfig file with authorization and master location information.")
	fs.StringVar(&s.Master, "master", s.Master, "The address of the Kubernetes API server (overrides any value in kubeconfig).")
	fs.StringVar(&s.Namespace, "namespace", metav1.NamespaceAll, "The namespace for which the MySQL operator manages MySQL clusters. Defaults to all.")
	fs.StringVar(&s.Images.MySQLServerImage, "mysql-server-image", s.Images.MySQLServerImage, "The name of the target 'mysql-server' image. Defaults to: mysql/mysql-server.")
	fs.StringVar(&s.Images.MySQLAgentImage, "mysql-agent-image", s.Images.MySQLAgentImage, "The name of the target 'mysql-agent' image. Defaults to: iad.ocir.io/oracle/mysql-agent.")
	fs.DurationVar(&s.MinResyncPeriod.Duration, "min-resync-period", s.MinResyncPeriod.Duration, "The resync period in reflectors will be random between MinResyncPeriod and 2*MinResyncPeriod.")
	return fs
}
