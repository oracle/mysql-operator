// Copyright 2017 The mysql-operator Authors
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

package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/pflag"

	flags "k8s.io/apiserver/pkg/util/flag"
	"k8s.io/apiserver/pkg/util/logs"

	"github.com/oracle/mysql-operator/cmd/mysql-operator/app"
	"github.com/oracle/mysql-operator/cmd/mysql-operator/app/options"
	"github.com/oracle/mysql-operator/pkg/controllers/cluster"
	"github.com/oracle/mysql-operator/pkg/version"
)

const (
	metricsEndpoint = "0.0.0.0:8080"
)

func main() {
	fmt.Fprintf(os.Stderr, "Starting mysql-operator version %s\n", version.GetBuildVersion())
	opts := options.NewMySQLOperatorServer()
	opts.AddFlags(pflag.CommandLine)

	flags.InitFlags()
	logs.InitLogs()
	defer logs.FlushLogs()

	cluster.RegisterMetrics()
	http.Handle("/metrics", prometheus.Handler())
	go http.ListenAndServe(metricsEndpoint, nil)

	if err := app.Run(opts); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
