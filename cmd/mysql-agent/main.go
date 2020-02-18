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

package main

import (
	goflag "flag"
	"fmt"
	"os"

	cliflag "k8s.io/component-base/cli/flag"

	"k8s.io/kubectl/pkg/util/logs"

	"github.com/spf13/pflag"
	glog "k8s.io/klog"

	"github.com/oracle/mysql-operator/cmd/mysql-agent/app"
	agentopts "github.com/oracle/mysql-operator/pkg/options/agent"
	"github.com/oracle/mysql-operator/pkg/version"
)

func main() {
	fmt.Fprintf(os.Stderr, "Starting mysql-agent version %s\n", version.GetBuildVersion())

	opts := agentopts.NewMySQLAgentOpts()

	opts.AddFlags(pflag.CommandLine)
	pflag.CommandLine.SetNormalizeFunc(cliflag.WordSepNormalizeFunc)
	pflag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	pflag.Parse()
	goflag.CommandLine.Parse([]string{})

	logs.InitLogs()
	defer logs.FlushLogs()

	pflag.VisitAll(func(flag *pflag.Flag) {
		glog.V(2).Infof("FLAG: --%s=%q", flag.Name, flag.Value)
	})

	if err := app.Run(opts); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
