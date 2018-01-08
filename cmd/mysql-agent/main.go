/*
mysql-backup-controller runs in a sidecar on each MySQL cluster pod and
performs backups by monitoring for MySQLBackup CRs associated with the cluster.
*/
package main

import (
	"fmt"
	"os"

	flags "k8s.io/apiserver/pkg/util/flag"
	"k8s.io/apiserver/pkg/util/logs"

	"github.com/spf13/pflag"

	"github.com/oracle/mysql-operator/cmd/mysql-agent/app"
	"github.com/oracle/mysql-operator/cmd/mysql-agent/app/options"
	"github.com/oracle/mysql-operator/pkg/version"
)

func main() {
	fmt.Fprintf(os.Stderr, "Starting mysql-agent version %s\n", version.GetBuildVersion())
	opts := options.NewMySQLAgentOpts()
	opts.AddFlags(pflag.CommandLine)

	flags.InitFlags()
	logs.InitLogs()
	defer logs.FlushLogs()

	if err := app.Run(opts); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
