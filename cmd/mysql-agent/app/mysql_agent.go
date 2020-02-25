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

package app

import (
	"context"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/heptiolabs/healthcheck"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	glog "k8s.io/klog"

	kubeinformers "k8s.io/client-go/informers"
	kubernetes "k8s.io/client-go/kubernetes"
	scheme "k8s.io/client-go/kubernetes/scheme"
	rest "k8s.io/client-go/rest"

	cluster "github.com/oracle/mysql-operator/pkg/cluster"
	backupcontroller "github.com/oracle/mysql-operator/pkg/controllers/backup"
	clustermgr "github.com/oracle/mysql-operator/pkg/controllers/cluster/manager"
	restorecontroller "github.com/oracle/mysql-operator/pkg/controllers/restore"
	clientset "github.com/oracle/mysql-operator/pkg/generated/clientset/versioned"
	opscheme "github.com/oracle/mysql-operator/pkg/generated/clientset/versioned/scheme"
	informers "github.com/oracle/mysql-operator/pkg/generated/informers/externalversions"
	agentopts "github.com/oracle/mysql-operator/pkg/options/agent"
	metrics "github.com/oracle/mysql-operator/pkg/util/metrics"
	signals "github.com/oracle/mysql-operator/pkg/util/signals"
)

const (
	metricsEndpoint = "0.0.0.0:8080"
)

func init() {
	opscheme.AddToScheme(scheme.Scheme)
}

// resyncPeriod computes the time interval a shared informer waits before
// resyncing with the api server.
func resyncPeriod(opts *agentopts.MySQLAgentOpts) func() time.Duration {
	return func() time.Duration {
		factor := rand.Float64() + 1
		return time.Duration(float64(opts.MinResyncPeriod.Nanoseconds()) * factor)
	}
}

// Run runs the MySQL backup controller. It should never exit.
func Run(opts *agentopts.MySQLAgentOpts) error {
	kubeconfig, err := rest.InClusterConfig()
	if err != nil {
		return err
	}

	ctx, cancelFunc := context.WithCancel(context.Background())

	// Set up signals so we handle the first shutdown signal gracefully.
	signals.SetupSignalHandler(cancelFunc)

	// Set up healthchecks (liveness and readiness).
	checkInCluster, err := cluster.NewHealthCheck()
	if err != nil {
		glog.Fatal(err)
	}
	health := healthcheck.NewHandler()
	health.AddReadinessCheck("node-in-cluster", checkInCluster)
	go func() {
		glog.Fatal(http.ListenAndServe(
			net.JoinHostPort(opts.Address, strconv.Itoa(int(opts.HealthcheckPort))),
			health,
		))
	}()

	kubeclient := kubernetes.NewForConfigOrDie(kubeconfig)
	mysqlopClient := clientset.NewForConfigOrDie(kubeconfig)

	sharedInformerFactory := informers.NewFilteredSharedInformerFactory(mysqlopClient, 0, opts.Namespace, nil)
	kubeInformerFactory := kubeinformers.NewFilteredSharedInformerFactory(kubeclient, resyncPeriod(opts)(), opts.Namespace, nil)

	var wg sync.WaitGroup

	manager, err := clustermgr.NewLocalClusterManger(kubeclient, kubeInformerFactory)
	if err != nil {
		return errors.Wrap(err, "failed to create new local MySQL InnoDB cluster manager")
	}

	// Initialise the agent metrics.
	metrics.RegisterPodName(opts.Hostname)
	metrics.RegisterClusterName(manager.Instance.ClusterName)
	clustermgr.RegisterMetrics()
	backupcontroller.RegisterMetrics()
	restorecontroller.RegisterMetrics()
	http.Handle("/metrics", prometheus.Handler())
	go http.ListenAndServe(metricsEndpoint, nil)

	// Block until local instance successfully initialised.
	for !manager.Sync(ctx) {
		time.Sleep(10 * time.Second)
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		manager.Run(ctx)
	}()

	backupController := backupcontroller.NewAgentController(
		kubeclient,
		mysqlopClient.MySQLV1alpha1(),
		sharedInformerFactory.MySQL().V1alpha1().Backups(),
		sharedInformerFactory.MySQL().V1alpha1().Clusters(),
		kubeInformerFactory.Core().V1().Pods(),
		opts.Hostname,
	)
	wg.Add(1)
	go func() {
		defer wg.Done()
		backupController.Run(ctx, 5)
	}()

	restoreController := restorecontroller.NewAgentController(
		kubeclient,
		mysqlopClient.MySQLV1alpha1(),
		sharedInformerFactory.MySQL().V1alpha1().Restores(),
		sharedInformerFactory.MySQL().V1alpha1().Clusters(),
		sharedInformerFactory.MySQL().V1alpha1().Backups(),
		kubeInformerFactory.Core().V1().Pods(),
		opts.Hostname,
	)
	wg.Add(1)
	go func() {
		defer wg.Done()
		restoreController.Run(ctx, 5)
	}()

	// Shared informers have to be started after ALL controllers.
	go sharedInformerFactory.Start(ctx.Done())
	go kubeInformerFactory.Start(ctx.Done())

	<-ctx.Done()

	glog.Info("Waiting for all controllers to shut down gracefully")
	wg.Wait()

	return nil
}
