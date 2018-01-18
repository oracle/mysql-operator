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

package framework

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	mysqlop "github.com/oracle/mysql-operator/pkg/generated/clientset/versioned"
)

// Global framework.
var Global *Framework

// Framework handles communication with the kube cluster in e2e tests.
type Framework struct {
	KubeClient          kubernetes.Interface
	MySQLOpClient       mysqlop.Interface
	Namespace           string
	BuildVersion        string
	SSHUser             string
	SSHKeyPath          string
	DestroyAfterFailure bool
}

// Setup sets up a test framework and initialises framework.Global.
func Setup() error {

	fmt.Printf("init> initKube...\n")
	// init kube clients
	kubeConfig := flag.String("kubeconfig", "", "Path to kubeconfig file with authorization and master location information.")
	namespace := flag.String("namespace", "default", "e2e test namespace")
	flag.Parse()
	fmt.Printf("init> initKube: build config\n")
	cfg, err := clientcmd.BuildConfigFromFlags("", *kubeConfig)
	if err != nil {
		return err
	}
	fmt.Printf("init> initKube: build kubeClient\n")
	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return err
	}
	fmt.Printf("init> initKube: build mysqlopClient\n")
	mysqlopClient, err := mysqlop.NewForConfig(cfg)
	if err != nil {
		return err
	}

	// init control variables
	var debug = false
	debugEnvStr, ok := os.LookupEnv("E2E_DEBUG")
	if ok {
		debug = strings.ToLower(debugEnvStr) == "true"
	}

	fmt.Printf("init> initFramework...\n")
	// init global framwork
	Global = &Framework{
		KubeClient:          kubeClient,
		MySQLOpClient:       mysqlopClient,
		Namespace:           *namespace,
		BuildVersion:        os.Getenv("MYSQL_OPERATOR_VERSION"),
		SSHUser:             "opc",
		SSHKeyPath:          os.Getenv("CLUSTER_INSTANCE_SSH_KEY"),
		DestroyAfterFailure: !debug,
	}
	return nil
}

// Teardown shuts down the test framework and cleans up.
func Teardown() error {
	// TODO: wait for all resources deleted.
	Global = nil
	return nil
}
