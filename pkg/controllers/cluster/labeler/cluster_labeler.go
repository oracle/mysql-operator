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

package labeler

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	glog "k8s.io/klog"

	corev1 "k8s.io/api/core/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	wait "k8s.io/apimachinery/pkg/util/wait"
	corev1informers "k8s.io/client-go/informers/core/v1"
	kubernetes "k8s.io/client-go/kubernetes"
	corev1listers "k8s.io/client-go/listers/core/v1"
	cache "k8s.io/client-go/tools/cache"
	workqueue "k8s.io/client-go/util/workqueue"

	cluster "github.com/oracle/mysql-operator/pkg/cluster"
	innodb "github.com/oracle/mysql-operator/pkg/cluster/innodb"
	constants "github.com/oracle/mysql-operator/pkg/constants"
	clusterctrl "github.com/oracle/mysql-operator/pkg/controllers/cluster"
	controllerutils "github.com/oracle/mysql-operator/pkg/controllers/util"
)

const controllerAgentName = "innodb-cluster-labeler"

// ClusterLabelerController adds annotations about the InnoDB cluster state
// to the Cluster's Pods. This controller should only be run iff the the
// local MySQL instance believes that it is the primary of the MySQL cluster.
type ClusterLabelerController struct {
	// localInstance represents the local MySQL instance.
	localInstance *cluster.Instance

	// podLister is able to list/get Pods from a shared informer's store.
	podLister corev1listers.PodLister
	// podListerSynced returns true if the Pod shared informer has synced at
	// least once.
	podListerSynced cache.InformerSynced
	// podControl enables control of cluster Pods.
	podControl clusterctrl.PodControlInterface

	queue workqueue.RateLimitingInterface
	store cache.Store
}

func keyFunc(obj interface{}) (string, error) {
	status, ok := obj.(*innodb.ClusterStatus)
	if !ok {
		return "", fmt.Errorf("expected *innodb.ClusterStatus got %T", obj)
	}
	return status.ClusterName, nil
}

// NewClusterLabelerController creates a new ClusterLabelerController.
func NewClusterLabelerController(
	localInstance *cluster.Instance,
	kubeClient kubernetes.Interface,
	podInformer corev1informers.PodInformer,
) *ClusterLabelerController {
	controller := &ClusterLabelerController{
		localInstance:   localInstance,
		podLister:       podInformer.Lister(),
		podListerSynced: podInformer.Informer().HasSynced,
		podControl:      clusterctrl.NewRealPodControl(kubeClient, podInformer.Lister()),
		queue:           workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), controllerAgentName),
		store:           cache.NewStore(keyFunc),
	}
	return controller
}

func (clc *ClusterLabelerController) updateClusterRoleLabel(pod *corev1.Pod, val string) error {
	new := pod.DeepCopy()
	if val == "" {
		delete(new.Labels, constants.LabelClusterRole)
	} else {
		new.Labels[constants.LabelClusterRole] = val
	}
	return clc.podControl.PatchPod(pod, new)
}

// syncHandler labels the Pods in a Cluster as being either a primary or
// secondary based on the given innodb.ClusterStatus.
func (clc *ClusterLabelerController) syncHandler(key string) error {
	obj, exists, err := clc.store.GetByKey(key)
	if err != nil {
		return errors.Wrapf(err, "getting key %q out of store", key)
	}
	if !exists {
		utilruntime.HandleError(fmt.Errorf("key %q in work queue no longer exists", key))
		return nil
	}
	status := obj.(*innodb.ClusterStatus)

	namespace := clc.localInstance.Namespace
	clusterName := clc.localInstance.ClusterName

	// Get any Pods already labeled as primaries for this cluster.
	primaries, err := clc.podLister.Pods(namespace).List(PrimarySelector(clusterName))
	if err != nil {
		return errors.Wrap(err, "failed to list primaries")
	}

	// Remove the mysql.oracle.com/role=primary label from any Pods that aren't
	// the local primary.
	primaryLabeled := false
	for _, pod := range primaries {
		if pod.Name == clc.localInstance.PodName() {
			primaryLabeled = true
			continue
		}

		var role string
		if !inCluster(status, pod.Name, clc.localInstance.Port) {
			glog.Infof("Removing %q label from previously labeled primary %s/%s",
				constants.LabelClusterRole, pod.Namespace, pod.Name)
			role = ""
		} else {
			glog.Infof("Labeling previously labeled primary %s/%s as secondary", pod.Namespace, pod.Name)
			role = constants.ClusterRoleSecondary
		}

		if err := clc.updateClusterRoleLabel(pod, role); err != nil {
			return errors.Wrap(err, "relabeling primary")
		}
	}

	// If the local primary is not yet labeled mysql.oracle.com/role=primary
	// label it.
	if !primaryLabeled {
		primary, err := clc.podLister.Pods(namespace).Get(clc.localInstance.PodName())
		if err != nil {
			return errors.Wrap(err, "failed to get primary Pod")
		}

		glog.Infof("Labeling %s/%s as primary", primary.Namespace, primary.Name)
		if err := clc.updateClusterRoleLabel(primary, constants.ClusterRolePrimary); err != nil {
			return errors.Wrapf(err, "labeling %s/%s as primary", primary.Namespace, primary.Name)
		}
	}

	// Get all non-primary Pods.
	pods, err := clc.podLister.Pods(namespace).List(NonPrimarySelector(clusterName))
	if err != nil {
		return errors.Wrap(err, "failed to list non-primary Cluster pods")
	}

	// Ensure they are labeled as secondary or not at all.
	for _, pod := range pods {
		if !inCluster(status, pod.Name, clc.localInstance.Port) {
			if HasRoleSelector(clusterName).Matches(labels.Set(pod.Labels)) {
				glog.Infof("Removing %q label from %s/%s as it's no longer in an ONLINE state",
					constants.LabelClusterRole, pod.Namespace, pod.Name)
				if err := clc.updateClusterRoleLabel(pod, ""); err != nil {
					return errors.Wrapf(err, "removing %q label from %s/%s", constants.LabelClusterRole, pod.Namespace, pod.Name)
				}
			}
			continue
		}
		if pod.Name != clc.localInstance.PodName() && !SecondarySelector(clusterName).Matches(labels.Set(pod.Labels)) {
			glog.Infof("Labeling %s/%s as secondary", pod.Namespace, pod.Name)
			if err := clc.updateClusterRoleLabel(pod, constants.ClusterRoleSecondary); err != nil {
				return errors.Wrapf(err, "labeling %s/%s as secondary", pod.Namespace, pod.Name)
			}
		}
	}

	return nil
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (clc *ClusterLabelerController) processNextWorkItem() bool {
	obj, shutdown := clc.queue.Get()
	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer clc.queue.Done().
	err := func(obj interface{}) error {
		defer clc.queue.Done(obj)
		key := obj.(string)
		if err := clc.syncHandler(key); err != nil {
			return errors.Wrapf(err, "error syncing %q", key)
		}

		clc.queue.Forget(obj)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(fmt.Errorf("Error syncing cluster status: %+v", err))
		clc.queue.AddRateLimited(obj)
	}

	return true
}

// runWorker runs a worker goroutine that invokes processNextWorkItem until the
// controller's queue is closed.
func (clc *ClusterLabelerController) runWorker() {
	for clc.processNextWorkItem() {
	}
}

// EnqueueClusterStatus takes an *innodb.ClusterStatus, stores it in the
// cache.Store, and then enqueues its key.
func (clc *ClusterLabelerController) EnqueueClusterStatus(obj interface{}) error {
	key, err := keyFunc(obj)
	if err != nil {
		return err
	}
	if err := clc.store.Add(obj); err != nil {
		return errors.Wrap(err, "adding cluster status to store")
	}
	clc.queue.Add(key)
	return nil
}

// Run runs the ClusterLabelerController.
func (clc *ClusterLabelerController) Run(ctx context.Context) {
	defer utilruntime.HandleCrash()
	defer clc.queue.ShutDown()

	glog.Infof("Starting the ClusterLabelerController")

	// Wait for the caches to be synced before starting worker
	glog.Info("Waiting for ClusterLabelerController informer caches to sync")
	if !controllerutils.WaitForCacheSync(controllerAgentName, ctx.Done(), clc.podListerSynced) {
		return
	}

	glog.Info("Starting ClusterLabelerController controller worker")
	go wait.Until(clc.runWorker, time.Second, ctx.Done())

	glog.Info("Started ClusterLabelerController controller worker")
	defer glog.Info("Shutting down ClusterLabelerController controller worker")
	<-ctx.Done()
}

// inCluster returns true if an instance is a functioning member of the InnoDB
// cluster.
func inCluster(status *innodb.ClusterStatus, podName string, port int) bool {
	statefuSetName, _ := cluster.GetParentNameAndOrdinal(podName)
	address := fmt.Sprintf("%s.%s:%d", podName, statefuSetName, port)
	inst, ok := status.DefaultReplicaSet.Topology[address]
	r := ok && (inst.Status == innodb.InstanceStatusOnline)
	return r
}
