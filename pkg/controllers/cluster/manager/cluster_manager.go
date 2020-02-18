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

package manager

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	glog "k8s.io/klog"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	wait "k8s.io/apimachinery/pkg/util/wait"
	kubeinformers "k8s.io/client-go/informers"
	kubernetes "k8s.io/client-go/kubernetes"
	utilexec "k8s.io/utils/exec"

	"github.com/oracle/mysql-operator/pkg/cluster"
	"github.com/oracle/mysql-operator/pkg/cluster/innodb"
	"github.com/oracle/mysql-operator/pkg/controllers/cluster/labeler"
	"github.com/oracle/mysql-operator/pkg/util/metrics"
	"github.com/oracle/mysql-operator/pkg/util/mysqlsh"
)

const pollingIntervalSeconds = 15

// ClusterManager manages the local MySQL instance's membership of an InnoDB cluster.
type ClusterManager struct {
	kubeClient kubernetes.Interface

	// kubeInformerFactory is a kubernetes core informer factory.
	kubeInformerFactory kubeinformers.SharedInformerFactory

	// mysqlshFactory creates new mysqlsh.Interfaces. Implemented as a factory
	// for testing purposes.
	mysqlshFactory func(uri string) mysqlsh.Interface

	// localMySh is a mysqlsh.Interface configured for the local instance of MySQL.
	localMySh mysqlsh.Interface

	// Instance is the local instance of MySQL under management.
	Instance *cluster.Instance

	// primaryCancelFunc cancels the execution of the primary-only controllers.
	primaryCancelFunc    context.CancelFunc
	podLabelerController *labeler.ClusterLabelerController
}

// NewClusterManager creates a InnoDB cluster ClusterManager.
func NewClusterManager(
	kubeClient kubernetes.Interface,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	mysqlshFactory func(string) mysqlsh.Interface,
	instance *cluster.Instance,
) *ClusterManager {
	manager := &ClusterManager{
		kubeClient:          kubeClient,
		kubeInformerFactory: kubeInformerFactory,
		mysqlshFactory:      mysqlshFactory,
		Instance:            instance,
		localMySh:           mysqlshFactory(instance.GetShellURI()),
	}
	return manager
}

// NewLocalClusterManger creates a new cluster.ClusterManager for the local MySQL instance.
func NewLocalClusterManger(kubeclient kubernetes.Interface, kubeInformerFactory kubeinformers.SharedInformerFactory) (*ClusterManager, error) {
	// Create a new instance representing the local MySQL instance.
	instance, err := cluster.NewLocalInstance()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get local MySQL instance")
	}

	return NewClusterManager(
		kubeclient,
		kubeInformerFactory,
		func(uri string) mysqlsh.Interface { return mysqlsh.New(utilexec.New(), uri) },
		instance,
	), nil
}

func (m *ClusterManager) getClusterStatus(ctx context.Context) (*innodb.ClusterStatus, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	clusterStatus, localMSHErr := m.localMySh.GetClusterStatus(ctx)
	if localMSHErr != nil {
		var err error
		clusterStatus, err = getClusterStatusFromGroupSeeds(ctx, m.kubeClient, m.Instance)
		if err != nil {
			// NOTE: We return the localMSHErr rather than the error here so that we
			// can dispatch on it.
			return nil, errors.Wrap(localMSHErr, "getting cluster status from group seeds")
		}
	}
	return clusterStatus, nil
}

// Sync ensures that the MySQL database instance managed by this instance of the
// agent is part of the InnoDB cluster and is online.
func (m *ClusterManager) Sync(ctx context.Context) bool {
	if !isDatabaseRunning(ctx) {
		glog.V(2).Infof("Database not yet running. Waiting...")
		return false
	}

	// First try the local instance so we reuse the mysqlsh.Instance in the
	// most common case.
	clusterStatus, err := m.getClusterStatus(ctx)
	if err != nil {
		myshErr, ok := errors.Cause(err).(*mysqlsh.Error)
		if !ok {
			glog.Errorf("Failed to get the cluster status: %+v", err)
			return false
		}

		// We can't find a cluster. Bootstrap if we're the first member of the
		// StatefulSet.
		if m.Instance.Ordinal == 0 {
			clusterStatus, err = m.bootstrap(ctx, myshErr)
			if err != nil {
				glog.Errorf("Error bootstrapping cluster: %v", err)
				metrics.IncEventCounter(clusterCreateErrorCount)
				return false
			}
			metrics.IncEventCounter(clusterCreateCount)
		} else {
			glog.V(2).Info("Cluster not yet present. Waiting...")
			return false
		}
	}

	// Set the cluster status so that the in-cluster healthcheck gets the
	// most up to date information.
	cluster.SetStatus(clusterStatus)

	if clusterStatus.DefaultReplicaSet.Status == innodb.ReplicaSetStatusNoQuorum {
		glog.V(4).Info("Cluster as seen from this instance is in NO_QUORUM state")
		metrics.IncEventCounter(clusterNoQuorumCount)
	}

	online := false
	instanceStatus := clusterStatus.GetInstanceStatus(m.Instance.Name())
	switch instanceStatus {
	case innodb.InstanceStatusOnline:
		metrics.IncStatusCounter(instanceStatusCount, innodb.InstanceStatusOnline)
		glog.V(4).Info("MySQL instance is online")
		online = true

	case innodb.InstanceStatusRecovering:
		metrics.IncStatusCounter(instanceStatusCount, innodb.InstanceStatusRecovering)
		glog.V(4).Info("MySQL instance is recovering")

	case innodb.InstanceStatusMissing:
		metrics.IncStatusCounter(instanceStatusCount, innodb.InstanceStatusMissing)
		primaryAddr, err := clusterStatus.GetPrimaryAddr()
		if err != nil {
			glog.Errorf("%v", err)
			return false
		}
		online = m.handleInstanceMissing(ctx, primaryAddr)
		if online {
			metrics.IncEventCounter(instanceRejoinCount)
		} else {
			metrics.IncEventCounter(instanceRejoinErrorCount)
		}

	case innodb.InstanceStatusNotFound:
		metrics.IncStatusCounter(instanceStatusCount, innodb.InstanceStatusNotFound)
		primaryAddr, err := clusterStatus.GetPrimaryAddr()
		if err != nil {
			glog.Errorf("%v", err)
			return false
		}
		online = m.handleInstanceNotFound(ctx, primaryAddr)
		if online {
			metrics.IncEventCounter(instanceAddCount)
		} else {
			metrics.IncEventCounter(instanceAddErrorCount)
		}

	case innodb.InstanceStatusUnreachable:
		metrics.IncStatusCounter(instanceStatusCount, innodb.InstanceStatusUnreachable)

	default:
		metrics.IncStatusCounter(instanceStatusCount, innodb.InstanceStatusUnknown)
		glog.Errorf("Received unrecognised cluster membership status: %q", instanceStatus)
	}

	if online && !m.Instance.MultiMaster {
		m.ensurePrimaryControllerState(ctx, clusterStatus)
	}

	return online
}

// ensurePrimaryControllerState ensures that the primary-only controllers are
// running if the local MySQL instance is the primary.
func (m *ClusterManager) ensurePrimaryControllerState(ctx context.Context, status *innodb.ClusterStatus) {
	// Are we the primary?
	primaryAddr, err := status.GetPrimaryAddr()
	if err != nil {
		glog.Errorf("%v", err)
		return
	}
	if !strings.HasPrefix(primaryAddr, m.Instance.Name()) {
		if m.primaryCancelFunc != nil {
			glog.V(4).Info("Calling primaryCancelFunc()")
			m.primaryCancelFunc()
			m.primaryCancelFunc = nil
		}
		return
	}

	// We are the Primary. Is/are the primary controller(s) running?
	if m.primaryCancelFunc == nil {
		// Run the primary controller(s).
		m.podLabelerController = labeler.NewClusterLabelerController(m.Instance, m.kubeClient, m.kubeInformerFactory.Core().V1().Pods())
		ctx, m.primaryCancelFunc = context.WithCancel(ctx)
		go m.podLabelerController.Run(ctx)
		// We must call Start() on the shared informer factory here to register
		// the new informer in the case of failover (where the shared informer
		// factory will have been started previously with no reference to the
		// Pod informer required by the labeler).
		go m.kubeInformerFactory.Start(ctx.Done())
	}

	if err := m.podLabelerController.EnqueueClusterStatus(status.DeepCopy()); err != nil {
		utilruntime.HandleError(errors.Wrap(err, "enqueuing ClusterStatus"))
	}
}

func (m *ClusterManager) handleInstanceMissing(ctx context.Context, primaryAddr string) bool {
	primaryURI := fmt.Sprintf("%s:%s@%s", m.Instance.GetUser(), m.Instance.GetPassword(), primaryAddr)
	primarySh := m.mysqlshFactory(primaryURI)

	// TODO: just call RejoinInstanceToCluster and handle the error.
	instanceState, err := primarySh.CheckInstanceState(ctx, m.Instance.GetShellURI())
	if err != nil {
		glog.Errorf("Failed to determine if we can rejoin the cluster: %v", err)
		return false
	}
	glog.V(4).Infof("Checking if instance can rejoin cluster")
	if instanceState.CanRejoinCluster() {
		whitelistCIDR, err := m.Instance.WhitelistCIDR()
		if err != nil {
			glog.Errorf("Getting CIDR to whitelist for GR: %v", err)
			return false
		}
		glog.V(4).Infof("Attempting to rejoin instance to cluster")
		if err := primarySh.RejoinInstanceToCluster(ctx, m.Instance.GetShellURI(), mysqlsh.Options{
			"ipWhitelist":   whitelistCIDR,
			"memberSslMode": "REQUIRED",
		}); err != nil {
			glog.Errorf("Failed to rejoin cluster: %v", err)
			return false
		}
	} else {
		glog.V(4).Infof("Removing instance from cluster")
		if err := primarySh.RemoveInstanceFromCluster(ctx, m.Instance.GetShellURI(), mysqlsh.Options{"force": "True"}); err != nil {
			glog.Errorf("Failed to remove from cluster: %v", err)
			return false
		}
	}
	return true
}

func (m *ClusterManager) handleInstanceNotFound(ctx context.Context, primaryAddr string) bool {
	glog.V(4).Infof("Adding secondary instance to the cluster")

	primaryURI := fmt.Sprintf("%s:%s@%s", m.Instance.GetUser(), m.Instance.GetPassword(), primaryAddr)
	psh := m.mysqlshFactory(primaryURI)

	whitelistCIDR, err := m.Instance.WhitelistCIDR()
	if err != nil {
		glog.Errorf("Getting CIDR to whitelist for GR: %v", err)
		return false
	}

	if err := psh.AddInstanceToCluster(ctx, m.Instance.GetShellURI(), mysqlsh.Options{
		"memberSslMode": "REQUIRED",
		"ipWhitelist":   whitelistCIDR,
	}); err != nil {
		glog.Errorf("Failed to add to cluster: %v", err)
		return false
	}
	return true
}

// bootstrap bootstraps the cluster. Called on the first Pod in the StatefulSet.
func (m *ClusterManager) bootstrap(ctx context.Context, mshErr *mysqlsh.Error) (*innodb.ClusterStatus, error) {
	if strings.Contains(mshErr.Message, "Cannot perform operation while group replication is starting up") {
		return nil, mshErr
	}

	if strings.Contains(mshErr.Message, "(metadata exists, but GR is not active)") {
		return m.rebootFromOutage(ctx)
	}

	return m.createCluster(ctx)
}

func (m *ClusterManager) createCluster(ctx context.Context) (*innodb.ClusterStatus, error) {
	glog.Infof("Creating InnoDB cluster")

	msh := m.mysqlshFactory(m.Instance.GetShellURI())

	whitelistCIDR, err := m.Instance.WhitelistCIDR()
	if err != nil {
		return nil, errors.Wrap(err, "getting CIDR to whitelist for  GR")
	}
	opts := mysqlsh.Options{
		"memberSslMode": "REQUIRED",
		"ipWhitelist":   whitelistCIDR,
	}
	if m.Instance.MultiMaster {
		opts["force"] = "True"
		opts["multiMaster"] = "True"
	}
	status, err := msh.CreateCluster(ctx, opts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create new cluster")
	}
	return status, nil
}

func (m *ClusterManager) rebootFromOutage(ctx context.Context) (*innodb.ClusterStatus, error) {
	glog.Info("Found existing InnoDB cluster (metadata exists, but GR is not active)")

	msh := m.mysqlshFactory(m.Instance.GetShellURI())
	if err := msh.RebootClusterFromCompleteOutage(ctx); err != nil {
		return nil, errors.Wrap(err, "rebooting cluster from complete outage")
	}

	status, err := msh.GetClusterStatus(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "getting cluster status")
	}
	return status, nil
}

// Run runs the ClusterManager controller.
// NOTE: ctx is not currently used for cancellation by caller (the stopCh is).
func (m *ClusterManager) Run(ctx context.Context) {
	wait.Until(func() { m.Sync(ctx) }, time.Second*pollingIntervalSeconds, ctx.Done())

	<-ctx.Done()

	// Stop the primary-only controllers if they're running
	if m.primaryCancelFunc != nil {
		m.primaryCancelFunc()
		m.primaryCancelFunc = nil
	}
}
