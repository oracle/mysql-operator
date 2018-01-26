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

package cluster

import (
	"context"
	"fmt"
	"strconv"
	"time"

	apps "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	wait "k8s.io/apimachinery/pkg/util/wait"
	version "k8s.io/apimachinery/pkg/version"
	appsinformers "k8s.io/client-go/informers/apps/v1beta1"
	coreinformers "k8s.io/client-go/informers/core/v1"
	kubernetes "k8s.io/client-go/kubernetes"
	scheme "k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	appslisters "k8s.io/client-go/listers/apps/v1beta1"
	corelisters "k8s.io/client-go/listers/core/v1"
	cache "k8s.io/client-go/tools/cache"
	record "k8s.io/client-go/tools/record"
	workqueue "k8s.io/client-go/util/workqueue"

	"github.com/golang/glog"
	"github.com/pkg/errors"

	api "github.com/oracle/mysql-operator/pkg/apis/mysql/v1"
	constants "github.com/oracle/mysql-operator/pkg/constants"
	controllerutils "github.com/oracle/mysql-operator/pkg/controllers/util"
	mysqlop "github.com/oracle/mysql-operator/pkg/generated/clientset/versioned"
	opscheme "github.com/oracle/mysql-operator/pkg/generated/clientset/versioned/scheme"
	opinformers "github.com/oracle/mysql-operator/pkg/generated/informers/externalversions/mysql/v1"
	oplisters "github.com/oracle/mysql-operator/pkg/generated/listers/mysql/v1"

	options "github.com/oracle/mysql-operator/cmd/mysql-operator/app/options"
	secrets "github.com/oracle/mysql-operator/pkg/resources/secrets"
	services "github.com/oracle/mysql-operator/pkg/resources/services"
	statefulsets "github.com/oracle/mysql-operator/pkg/resources/statefulsets"
	metrics "github.com/oracle/mysql-operator/pkg/util/metrics"
	buildversion "github.com/oracle/mysql-operator/pkg/version"
)

const controllerAgentName = "mysql-operator"

const (
	// SuccessSynced is used as part of the Event 'reason' when a MySQSL is
	// synced.
	SuccessSynced = "Synced"
	// ErrResourceExists is used as part of the Event 'reason' when a
	// MySQLCluster fails to sync due to a resource of the same name already
	// existing.
	ErrResourceExists = "ErrResourceExists"

	// MessageResourceExists is the message used for Events when a resource
	// fails to sync due to a resource already existing.
	MessageResourceExists = "%s %s/%s already exists and is not managed by MySQLCluster"
	// MessageResourceSynced is the message used for an Event fired when a
	// MySQLCluster is synced successfully
	MessageResourceSynced = "MySQLCluster synced successfully"
)

// The MySQLController watches the Kubernetes API for changes to MySQL resources
type MySQLController struct {
	// Global MySQLOperator configuration options.
	opConfig options.MySQLOperatorServer

	kubeClient kubernetes.Interface
	opClient   mysqlop.Interface

	shutdown bool
	queue    workqueue.RateLimitingInterface

	// clusterLister is able to list/get MySQLClusters from a shared informer's
	// store.
	clusterLister oplisters.MySQLClusterLister
	// clusterListerSynced returns true if the MySQLCluster shared informer has
	// synced at least once.
	clusterListerSynced cache.InformerSynced
	// clusterUpdater implements control logic for updating MySQLCluster
	// statuses. Implemented as an interface to enable testing.
	clusterUpdater clusterUpdaterInterface

	// statefulSetLister is able to list/get StatefulSets from a shared
	// informer's store.
	statefulSetLister appslisters.StatefulSetLister
	// statefulSetListerSynced returns true if the StatefulSet shared informer
	// has synced at least once.
	statefulSetListerSynced cache.InformerSynced
	// statefulSetControl enables control of StatefulSets associated with
	// MySQLClusters.
	statefulSetControl StatefulSetControlInterface

	// podLister is able to list/get Pods from a shared
	// informer's store.
	podLister corelisters.PodLister
	// podListerSynced returns true if the Pod shared informer
	// has synced at least once.
	podListerSynced cache.InformerSynced
	// podControl enables control of Pods associated with
	// MySQLClusters.
	podControl PodControlInterface

	// serviceLister is able to list/get Services from a shared informer's
	// store.
	serviceLister corelisters.ServiceLister
	// serviceListerSynced returns true if the Service shared informer
	// has synced at least once.
	serviceListerSynced cache.InformerSynced

	// serviceControl enables control of Services associated with MySQLClusters.
	serviceControl ServiceControlInterface

	// secretControl enables control of Services associated with MySQLClusters.
	secretControl SecretControlInterface

	// apiServerVersion holds version information about the Kubernetes API
	// server of the current cluster.
	apiServerVersion *version.Info

	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder

	//configControl enables control of the config associated with MySQLClusters.
	// TODO: 'configMapControl' would be more consistent?
	configControl ConfigMapControlInterface

	// configMapLister is able to list/get ConfigMaps from a shared
	// informer's store.
	configMapLister corelisters.ConfigMapLister

	// configMapListerSynced returns true if the ConfigMap shared informer
	// has synced at least once.
	configMapListerSynced cache.InformerSynced
}

// NewController creates a new MySQLController.
func NewController(
	opConfig options.MySQLOperatorServer,
	opClient mysqlop.Interface,
	kubeClient kubernetes.Interface,
	apiServerVersion *version.Info,
	clusterInformer opinformers.MySQLClusterInformer,
	statefulSetInformer appsinformers.StatefulSetInformer,
	podInformer coreinformers.PodInformer,
	serviceInformer coreinformers.ServiceInformer,
	configMapInformer coreinformers.ConfigMapInformer,
	resyncPeriod time.Duration,
	namespace string,
) *MySQLController {
	opscheme.AddToScheme(scheme.Scheme) // TODO: This shouldn't be done here I don't think.

	// Create event broadcaster.
	glog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	m := MySQLController{
		opConfig: opConfig,

		opClient:   opClient,
		kubeClient: kubeClient,

		clusterLister:       clusterInformer.Lister(),
		clusterListerSynced: clusterInformer.Informer().HasSynced,
		clusterUpdater:      newClusterUpdater(opClient, clusterInformer.Lister()),

		serviceLister:       serviceInformer.Lister(),
		serviceListerSynced: serviceInformer.Informer().HasSynced,
		serviceControl:      NewRealServiceControl(kubeClient, serviceInformer.Lister()),

		statefulSetLister:       statefulSetInformer.Lister(),
		statefulSetListerSynced: statefulSetInformer.Informer().HasSynced,
		statefulSetControl:      NewRealStatefulSetControl(kubeClient, statefulSetInformer.Lister()),

		podLister:       podInformer.Lister(),
		podListerSynced: podInformer.Informer().HasSynced,
		podControl:      NewRealPodControl(kubeClient, podInformer.Lister()),

		secretControl: NewRealSecretControl(kubeClient),

		queue:            workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "mysqlcluster"),
		apiServerVersion: apiServerVersion,
		recorder:         recorder,
	}

	clusterInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: m.enqueueCluster,
		UpdateFunc: func(old, new interface{}) {
			m.enqueueCluster(new)
		},
		DeleteFunc: func(obj interface{}) {
			major, _ := strconv.Atoi(m.apiServerVersion.Major)
			minor, _ := strconv.Atoi(m.apiServerVersion.Minor)
			if major <= 1 && minor <= 7 {
				if err := m.deleteClusterResources(obj); err != nil {
					utilruntime.HandleError(fmt.Errorf("Failed to delete cluster resources: %v", err))
				}
			}

			cluster, ok := obj.(*api.MySQLCluster)
			if ok {
				m.onClusterDeleted(cluster.Name)
			}
		},
	})

	statefulSetInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: m.handleObject,
		UpdateFunc: func(old, new interface{}) {
			newStatefulSet := new.(*apps.StatefulSet)
			oldStatefulSet := old.(*apps.StatefulSet)
			if newStatefulSet.ResourceVersion == oldStatefulSet.ResourceVersion {
				return
			}

			// If cluster is ready ...
			if newStatefulSet.Status.ReadyReplicas == newStatefulSet.Status.Replicas {
				clusterName, ok := newStatefulSet.Labels[constants.MySQLClusterLabel]
				if ok {
					m.onClusterReady(clusterName)
				}
			}
			m.handleObject(new)
		},
		DeleteFunc: m.handleObject,
	})

	m.configMapLister = configMapInformer.Lister()
	m.configMapListerSynced = statefulSetInformer.Informer().HasSynced
	m.configControl = NewRealConfigMapControl(kubeClient, m.configMapLister)
	return &m
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (m *MySQLController) Run(ctx context.Context, threadiness int) {
	defer utilruntime.HandleCrash()
	defer m.queue.ShutDown()

	glog.Info("Starting MySQLCluster controller")

	// Wait for the caches to be synced before starting workers
	glog.Info("Waiting for MySQLCluster controller informer caches to sync")
	if !controllerutils.WaitForCacheSync("mysql cluster", ctx.Done(),
		m.clusterListerSynced,
		m.statefulSetListerSynced,
		m.podListerSynced,
		m.serviceListerSynced,
		m.configMapListerSynced) {
		return
	}

	glog.Info("Starting MySQLCluster controller workers")
	// Launch two workers to process Foo resources
	for i := 0; i < threadiness; i++ {
		go wait.Until(m.runWorker, time.Second, ctx.Done())
	}

	glog.Info("Started MySQLCluster controller workers")
	defer glog.Info("Shutting down MySQLCluster controller workers")
	<-ctx.Done()
}

// worker runs a worker goroutine that invokes processNextWorkItem until the
// controller's queue is closed.
func (m *MySQLController) runWorker() {
	for m.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (m *MySQLController) processNextWorkItem() bool {
	obj, shutdown := m.queue.Get()

	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer m.queue.Done(obj)
		key, ok := obj.(string)
		if !ok {
			m.queue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in queue but got %#v", obj))
			return nil
		}
		if err := m.syncHandler(key); err != nil {
			return fmt.Errorf("error syncing '%s': %s", key, err.Error())
		}
		m.queue.Forget(obj)
		glog.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the MySQLCluster
// resource with the current status of the resource.
func (m *MySQLController) syncHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name.
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the MySQLCluster resource with this namespace/name.
	cluster, err := m.clusterLister.MySQLClusters(namespace).Get(name)
	if err != nil {
		// The MySQLCluster resource may no longer exist, in which case we stop
		// processing.
		if apierrors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("mysqlcluster '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	cluster.EnsureDefaults()
	if err = cluster.Validate(); err != nil {
		return err
	}

	// Ensure that the required labels are set on the cluster.
	if !SelectorForCluster(cluster).Matches(labels.Set(cluster.Labels)) {
		glog.V(2).Infof("Setting label on cluster %s", SelectorForCluster(cluster).String())
		if cluster.Labels == nil {
			cluster.Labels = make(map[string]string)
		}
		cluster.Labels[constants.MySQLClusterLabel] = cluster.Name
		cluster.Labels[constants.MySQLOperatorVersionLabel] = buildversion.GetBuildVersion()
		return m.clusterUpdater.UpdateClusterLabels(cluster.DeepCopy(), labels.Set(cluster.Labels))
	}

	// Create a MySQL root password secret for the cluster if required and one
	// does not already exist.
	if cluster.RequiresSecret() {
		_, err := m.secretControl.GetForCluster(cluster)
		if apierrors.IsNotFound(err) {
			glog.V(2).Infof("Creating a new root password Secret for cluster %s/%s", cluster.Namespace, cluster.Name)
			err = m.secretControl.CreateSecret(secrets.NewMysqlRootPassword(cluster))
		}
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("could not get secrets from cluster '%s/%s'", cluster.Namespace, cluster.Name))
		}
	}

	// TODO: Should create shared getter method in resources/services.

	svc, err := m.serviceLister.Services(cluster.Namespace).Get(cluster.Name)
	// If the resource doesn't exist, we'll create it
	if apierrors.IsNotFound(err) {
		glog.V(2).Infof("Creating a new Service for cluster %s/%s", cluster.Namespace, cluster.Name)

		svc = services.NewForCluster(cluster)
		err = m.serviceControl.CreateService(svc)
	}

	// If an error occurs during Get/Create, we'll requeue the item so we can
	// attempt processing again later. This could have been caused by a
	// temporary network failure, or any other transient reason.
	if err != nil {
		return err
	}

	// If the Service is not controlled by this MySQLCluster resource, we should
	// log a warning to the event recorder and return.
	if !metav1.IsControlledBy(svc, cluster) {
		msg := fmt.Sprintf(MessageResourceExists, "Service", svc.Namespace, svc.Name)
		m.recorder.Event(cluster, corev1.EventTypeWarning, ErrResourceExists, msg)
		return errors.New(msg)
	}

	// TODO: Should create shared getter method in resources/statefulsets.

	ss, err := m.statefulSetLister.StatefulSets(cluster.Namespace).Get(cluster.Name)
	// If the resource doesn't exist, we'll create it
	if apierrors.IsNotFound(err) {
		glog.V(2).Infof("Creating a new StatefulSet for cluster %s/%s", cluster.Namespace, cluster.Name)
		ss = statefulsets.NewForCluster(cluster, m.opConfig.Images, svc.Name)
		err = m.statefulSetControl.CreateStatefulSet(ss)
	}

	// If an error occurs during Get/Create, we'll requeue the item so we can
	// attempt processing again later. This could have been caused by a
	// temporary network failure, or any other transient reason.
	if err != nil {
		return err
	}

	// If the StatefulSet is not controlled by this MySQLCluster resource, we
	// should log a warning to the event recorder and return.
	if !metav1.IsControlledBy(ss, cluster) {
		msg := fmt.Sprintf(MessageResourceExists, "StatefulSet", ss.Namespace, ss.Name)
		m.recorder.Event(cluster, corev1.EventTypeWarning, ErrResourceExists, msg)
		return fmt.Errorf(msg)
	}

	// Upgrade the required component resources the current MySQLOperator version.
	err = m.ensureMySQLOperatorVersion(cluster, ss, buildversion.GetBuildVersion())
	if err != nil {
		return err
	}

	// If this number of the replicas on the MySQLCluster does not equal the
	// current desired replicas on the StatefulSet, we should update the
	// StatefulSet resource.
	if cluster.Spec.Replicas != *ss.Spec.Replicas {
		glog.V(4).Infof("Updating %q: clusterReplicas=%d statefulSetReplicas=%d",
			cluster.Spec.Replicas, ss.Spec.Replicas)
		ss = statefulsets.NewForCluster(cluster, m.opConfig.Images, svc.Name)
		ss, err = m.kubeClient.AppsV1beta1().StatefulSets(cluster.Namespace).Update(ss)
		// If an error occurs during Update, we'll requeue the item so we can
		// attempt processing again later. This could have been caused by a
		// temporary network failure, or any other transient reason.
		if err != nil {
			return err
		}
	}

	// Finally, we update the status block of the MySQLCluster resource to
	// reflect the current state of the world.
	err = m.updateClusterStatus(cluster, ss)
	if err != nil {
		return err
	}

	m.recorder.Event(cluster, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
	return nil
}

// ensureMySQLOperatorVersion updates the MySQLOperator resource types that require it to make it consistent with the specifed operator version.
func (m *MySQLController) ensureMySQLOperatorVersion(c *api.MySQLCluster, ss *apps.StatefulSet, operatorVersion string) error {
	// Ensure the Pods belonging to the MySQLCluster are updated to the correct 'mysql-agent' image for the current MySQLOperator version.
	container := statefulsets.MySQLAgentName
	pods, err := m.podLister.List(SelectorForCluster(c))
	for _, pod := range pods {
		if requiresMySQLAgentPodUpgrade(pod, container, operatorVersion) && canUpgradeMySQLAgent(pod) {
			glog.Infof("Upgrading cluster pod '%s/%s' to latest operator version: %s", pod.Namespace, pod.Name, operatorVersion)
			updated := updatePodToOperatorVersion(pod.DeepCopy(), m.opConfig.Images.MySQLAgentImage, operatorVersion)
			err = m.podControl.PatchPod(pod, updated)
			if err != nil {
				return errors.Wrap(err, "upgrade operator version: PatchPod failed")
			}
		}
	}

	// Ensure the StatefulSet is updated with the correct template 'mysql-agent' image for the current MySQLOperator version.
	if requiresMySQLAgentStatefulSetUpgrade(ss, container, operatorVersion) {
		glog.Infof("Upgrading cluster statefulset '%s/%s' to latest operator version: %s", ss.Namespace, ss.Name, operatorVersion)
		updated := updateStatefulSetToOperatorVersion(ss.DeepCopy(), m.opConfig.Images.MySQLAgentImage, operatorVersion)
		err = m.statefulSetControl.PatchStatefulSet(ss, updated)
		if err != nil {
			return errors.Wrap(err, "upgrade operator version: PatchStatefulSet failed")
		}
	}

	// Ensure the MySQLCluster is updated with the correct MySQLOperator version.
	if !SelectorForClusterOperatorVersion(operatorVersion).Matches(labels.Set(c.Labels)) {
		glog.Infof("Upgrading cluster statefulset '%s/%s' to latest operator version: %s", c.Namespace, c.Name, operatorVersion)
		copy := c.DeepCopy()
		copy.Labels[constants.MySQLOperatorVersionLabel] = operatorVersion
		err := m.clusterUpdater.UpdateClusterLabels(copy, labels.Set(copy.Labels))
		if err != nil {
			return errors.Wrap(err, "upgrade operator version: MySQLClusterUpdate failed")
		}
	}
	return nil
}

// updateClusterStatusForSS updates MySQLCluster statuses based on changes to their associated StatefulSets.
func (m *MySQLController) updateClusterStatus(cluster *api.MySQLCluster, ss *apps.StatefulSet) error {
	glog.V(4).Infof("%s/%s: ss.Spec.Replicas=%d, ss.Status.ReadyReplicas=%d, ss.Status.Replicas=%d",
		cluster.Namespace, cluster.Name, *ss.Spec.Replicas, ss.Status.ReadyReplicas, ss.Status.Replicas)

	phase := cluster.Status.Phase

	if (ss.Status.ReadyReplicas < ss.Status.Replicas) || (*ss.Spec.Replicas != ss.Status.Replicas) {
		phase = api.MySQLClusterPending
	} else if ss.Status.ReadyReplicas == ss.Status.Replicas {
		phase = api.MySQLClusterRunning
	}

	if phase != cluster.Status.Phase {
		status := cluster.Status.DeepCopy()
		status.Phase = phase
		if err := m.clusterUpdater.UpdateClusterStatus(cluster.DeepCopy(), status); err != nil {
			return fmt.Errorf("failed to update cluster status: %v", err)
		}
	}

	return nil
}

// deleteClusterResources manually issues a delete for the dependant resources
// of a MySQLCluster.
// DEPRECIATED: Not required after Kubernetes 1.8. Remove when dropping support
// for 1.7.
func (m *MySQLController) deleteClusterResources(obj interface{}) error {
	cluster, ok := obj.(*api.MySQLCluster)
	if !ok {
		d, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			return fmt.Errorf("deleteClusterResources() received unexpected type %T", obj)
		}
		// We don't care about the object being stale. We want to ensure that
		// its dependant resources are absent anyway.
		cluster = d.Obj.(*api.MySQLCluster)
	}

	if cluster.RequiresSecret() {
		glog.V(4).Infof("Ensuring Secret deleted for MySQLCLuster %s/%s", cluster.Namespace, cluster.Name)
		s, err := m.secretControl.GetForCluster(cluster)
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return err
			}
		} else if metav1.IsControlledBy(s, cluster) {
			if err := m.secretControl.DeleteSecret(s); err != nil {
				return err
			}
		}
	}

	// TODO(apryde): This needs to be modified to check for user-defined my.cnf
	// ConfigMap as by default one is no longer created.
	glog.V(4).Infof("Ensuring my.cnf ConfigMap deleted for MySQLCLuster %s/%s", cluster.Namespace, cluster.Name)
	mycnf, err := m.configMapLister.ConfigMaps(cluster.Namespace).Get(cluster.Name)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
	} else if metav1.IsControlledBy(mycnf, cluster) {
		if err := m.configControl.DeleteConfigMap(mycnf); err != nil {
			return err
		}
	}

	glog.V(4).Infof("Ensuring Service deleted for MySQLCLuster %s/%s", cluster.Namespace, cluster.Name)
	svcName := cluster.Name
	svc, err := m.serviceLister.Services(cluster.Namespace).Get(svcName)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
	} else if metav1.IsControlledBy(svc, cluster) {
		if err := m.serviceControl.DeleteService(svc); err != nil {
			return err
		}
	}

	// Ensure the cluster's StatefulSet does not exist.
	glog.V(4).Infof("Ensuring StatefulSet deleted for MySQLCLuster %s/%s", cluster.Namespace, cluster.Name)
	statefulSetName := cluster.Name
	ss, err := m.statefulSetLister.StatefulSets(cluster.Namespace).Get(statefulSetName)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
	} else if metav1.IsControlledBy(ss, cluster) {
		if err := m.statefulSetControl.DeleteStatefulSet(ss); err != nil {
			return err
		}
	}

	glog.V(4).Infof("Ensured all components of MySQLCLuster %s/%s deleted", cluster.Namespace, cluster.Name)

	return nil
}

// enqueueCluster takes a MySQLCluster resource and converts it into a
// namespace/name string which is then put onto the work queue. This method
// should *not* be passed resources of any type other than MySQLCluster.
func (m *MySQLController) enqueueCluster(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(err)
		return
	}
	m.queue.AddRateLimited(key)
}

// handleObject will take any resource implementing metav1.Object and attempt
// to find the MySQLResource that 'owns' it. It does this by looking at the
// objects metadata.ownerReferences field for an appropriate OwnerReference.
// It then enqueues that Foo resource to be processed. If the object does not
// have an appropriate OwnerReference, it will simply be skipped.
func (m *MySQLController) handleObject(obj interface{}) {
	object, ok := obj.(metav1.Object)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object, invalid type"))
			return
		}
		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object tombstone, invalid type"))
			return
		}
		glog.V(4).Infof("Recovered deleted object '%s' from tombstone", object.GetName())
	}

	glog.V(4).Infof("Processing object: %s", object.GetName())
	if ownerRef := metav1.GetControllerOf(object); ownerRef != nil {
		// If this object is not owned by a MySQLCluster, we should not do
		// anything more with it.
		if ownerRef.Kind != api.MySQLClusterCRDResourceKind {
			return
		}

		cluster, err := m.clusterLister.MySQLClusters(object.GetNamespace()).Get(ownerRef.Name)
		if err != nil {
			glog.V(4).Infof("ignoring orphaned object '%s' of MySQLCluster '%s'", object.GetSelfLink(), ownerRef.Name)
			return
		}

		m.enqueueCluster(cluster)
		return
	}
}

func (m *MySQLController) onClusterReady(clusterName string) {
	glog.V(2).Infof("Cluster %s ready", clusterName)
	metrics.IncEventCounter(clustersCreatedCount)
	metrics.IncEventGauge(clustersTotalCount)
}

func (m *MySQLController) onClusterDeleted(clusterName string) {
	glog.V(2).Infof("Cluster %s deleted", clusterName)
	metrics.IncEventCounter(clustersDeletedCount)
	metrics.DecEventGauge(clustersTotalCount)
}
