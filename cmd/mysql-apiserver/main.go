package main

import (
	"flag"
	"os"

	"github.com/golang/glog"
	"github.com/spf13/pflag"

	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/apiserver/pkg/util/logs"

	"github.com/oracle/mysql-operator/cmd/mysql-apiserver/app"
)

func main() {
	stopCh := genericapiserver.SetupSignalHandler()

	o := app.NewMySQLAPIServerOptions(os.Stdout, os.Stderr)
	o.AddFlags(pflag.CommandLine)

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	flag.CommandLine.Parse([]string{})

	logs.InitLogs()
	defer logs.FlushLogs()

	if err := app.Run(o, stopCh); err != nil {
		glog.Fatal(err)
	}
}
