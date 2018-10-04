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
	"strings"
	"time"

	apps "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	wait "k8s.io/apimachinery/pkg/util/wait"
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

	"github.com/coreos/go-semver/semver"
	"github.com/golang/glog"
	"github.com/pkg/errors"

	clusterutil "github.com/oracle/mysql-operator/pkg/api/cluster"
	v1alpha1 "github.com/oracle/mysql-operator/pkg/apis/mysql/v1alpha1"
	constants "github.com/oracle/mysql-operator/pkg/constants"
	controllerutils "github.com/oracle/mysql-operator/pkg/controllers/util"
	clientset "github.com/oracle/mysql-operator/pkg/generated/clientset/versioned"
	opscheme "github.com/oracle/mysql-operator/pkg/generated/clientset/versioned/scheme"
	informersv1alpha1 "github.com/oracle/mysql-operator/pkg/generated/informers/externalversions/mysql/v1alpha1"
	listersv1alpha1 "github.com/oracle/mysql-operator/pkg/generated/listers/mysql/v1alpha1"

	operatoropts "github.com/oracle/mysql-operator/pkg/options/operator"
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
	// Cluster fails to sync due to a resource of the same name already
	// existing.
	ErrResourceExists = "ErrResourceExists"

	// MessageResourceExists is the message used for Events when a resource
	// fails to sync due to a resource already existing.
	MessageResourceExists = "%s %s/%s already exists and is not managed by Cluster"
	// MessageResourceSynced is the message used for an Event fired when a
	// Cluster is synced successfully
	MessageResourceSynced = "Cluster synced successfully"
)

// The MySQLController watches the Kubernetes API for changes to MySQL resources
type MySQLController struct {
	// Global MySQLOperator configuration options.
	opConfig operatoropts.MySQLOperatorOpts

	kubeClient kubernetes.Interface
	opClient   clientset.Interface

	shutdown bool
	queue    workqueue.RateLimitingInterface

	// clusterLister is able to list/get Clusters from a shared informer's
	// store.
	clusterLister listersv1alpha1.ClusterLister
	// clusterListerSynced returns true if the Cluster shared informer has
	// synced at least once.
	clusterListerSynced cache.InformerSynced
	// clusterUpdater implements control logic for updating Cluster
	// statuses. Implemented as an interface to enable testing.
	clusterUpdater clusterUpdaterInterface

	// statefulSetLister is able to list/get StatefulSets from a shared
	// informer's store.
	statefulSetLister appslisters.StatefulSetLister
	// statefulSetListerSynced returns true if the StatefulSet shared informer
	// has synced at least once.
	statefulSetListerSynced cache.InformerSynced
	// statefulSetControl enables control of StatefulSets associated with
	// Clusters.
	statefulSetControl StatefulSetControlInterface

	// podLister is able to list/get Pods from a shared
	// informer's store.
	podLister corelisters.PodLister
	// podListerSynced returns true if the Pod shared informer
	// has synced at least once.
	podListerSynced cache.InformerSynced
	// podControl enables control of Pods associated with
	// Clusters.
	podControl PodControlInterface

	// serviceLister is able to list/get Services from a shared informer's
	// store.
	serviceLister corelisters.ServiceLister
	// serviceListerSynced returns true if the Service shared informer
	// has synced at least once.
	serviceListerSynced cache.InformerSynced

	// serviceControl enables control of Services associated with Clusters.
	serviceControl ServiceControlInterface

	// secretControl enables control of Services associated with Clusters.
	secretControl SecretControlInterface

	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder
}

// NewController creates a new MySQLController.
func NewController(
	opConfig operatoropts.MySQLOperatorOpts,
	opClient clientset.Interface,
	kubeClient kubernetes.Interface,
	clusterInformer informersv1alpha1.ClusterInformer,
	statefulSetInformer appsinformers.StatefulSetInformer,
	podInformer coreinformers.PodInformer,
	serviceInformer coreinformers.ServiceInformer,
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

		queue:    workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "mysqlcluster"),
		recorder: recorder,
	}

	clusterInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: m.enqueueCluster,
		UpdateFunc: func(old, new interface{}) {
			m.enqueueCluster(new)
		},
		DeleteFunc: func(obj interface{}) {
			cluster, ok := obj.(*v1alpha1.Cluster)
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
				clusterName, ok := newStatefulSet.Labels[constants.ClusterLabel]
				if ok {
					m.onClusterReady(clusterName)
				}
			}
			m.handleObject(new)
		},
		DeleteFunc: m.handleObject,
	})

	return &m
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (m *MySQLController) Run(ctx context.Context, threadiness int) {
	defer utilruntime.HandleCrash()
	defer m.queue.ShutDown()

	glog.Info("Starting Cluster controller")

	// Wait for the caches to be synced before starting workers
	glog.Info("Waiting for Cluster controller informer caches to sync")
	if !controllerutils.WaitForCacheSync("mysql cluster", ctx.Done(),
		m.clusterListerSynced,
		m.statefulSetListerSynced,
		m.podListerSynced,
		m.serviceListerSynced) {
		return
	}

	glog.Info("Starting Cluster controller workers")
	// Launch two workers to process Foo resources
	for i := 0; i < threadiness; i++ {
		go wait.Until(m.runWorker, time.Second, ctx.Done())
	}

	glog.Info("Started Cluster controller workers")
	defer glog.Info("Shutting down Cluster controller workers")
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
// converge the two. It then updates the Status block of the Cluster
// resource with the current status of the resource.
func (m *MySQLController) syncHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name.
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	nsName := types.NamespacedName{Namespace: namespace, Name: name}

	// Get the Cluster resource with this namespace/name.
	cluster, err := m.clusterLister.Clusters(namespace).Get(name)
	if err != nil {
		// The Cluster resource may no longer exist, in which case we stop processing.
		if apierrors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("mysqlcluster '%s' in work queue no longer exists", key))
			return nil
		}
		return err
	}

	cluster.EnsureDefaults()
	if err = cluster.Validate(); err != nil {
		return errors.Wrap(err, "validating Cluster")
	}

	if cluster.Spec.Repository == "" {
		cluster.Spec.Repository = m.opConfig.Images.DefaultMySQLServerImage
	}

	operatorVersion := buildversion.GetBuildVersion()
	// Ensure that the required labels are set on the cluster.
	sel := combineSelectors(SelectorForCluster(cluster), SelectorForClusterOperatorVersion(operatorVersion))
	if !sel.Matches(labels.Set(cluster.Labels)) {
		glog.V(2).Infof("Setting labels on cluster %s", SelectorForCluster(cluster).String())
		if cluster.Labels == nil {
			cluster.Labels = make(map[string]string)
		}
		cluster.Labels[constants.ClusterLabel] = cluster.Name
		cluster.Labels[constants.MySQLOperatorVersionLabel] = buildversion.GetBuildVersion()
		return m.clusterUpdater.UpdateClusterLabels(cluster.DeepCopy(), labels.Set(cluster.Labels))
	}

	// Create a MySQL root password secret for the cluster if required.
	if cluster.RequiresSecret() {
		err = m.secretControl.CreateSecret(secrets.NewMysqlRootPassword(cluster))
		if err != nil && !apierrors.IsAlreadyExists(err) {
			return errors.Wrap(err, "creating root password Secret")
		}
	}

	svc, err := m.serviceLister.Services(cluster.Namespace).Get(cluster.Name)
	// If the resource doesn't exist, we'll create it
	if apierrors.IsNotFound(err) {
		glog.V(2).Infof("Creating a new Service for cluster %q", nsName)
		svc = services.NewForCluster(cluster)
		err = m.serviceControl.CreateService(svc)
	}

	// If an error occurs during Get/Create, we'll requeue the item so we can
	// attempt processing again later. This could have been caused by a
	// temporary network failure, or any other transient reason.
	if err != nil {
		return err
	}

	// If the Service is not controlled by this Cluster resource, we should
	// log a warning to the event recorder and return.
	if !metav1.IsControlledBy(svc, cluster) {
		msg := fmt.Sprintf(MessageResourceExists, "Service", svc.Namespace, svc.Name)
		m.recorder.Event(cluster, corev1.EventTypeWarning, ErrResourceExists, msg)
		return errors.New(msg)
	}

	ss, err := m.statefulSetLister.StatefulSets(cluster.Namespace).Get(cluster.Name)
	// If the resource doesn't exist, we'll create it
	if apierrors.IsNotFound(err) {
		glog.V(2).Infof("Creating a new StatefulSet for cluster %q", nsName)
		ss = statefulsets.NewForCluster(cluster, m.opConfig.Images, svc.Name)
		err = m.statefulSetControl.CreateStatefulSet(ss)
	}

	// If an error occurs during Get/Create, we'll requeue the item so we can
	// attempt processing again later. This could have been caused by a
	// temporary network failure, or any other transient reason.
	if err != nil {
		return err
	}

	// If the StatefulSet is not controlled by this Cluster resource, we
	// should log a warning to the event recorder and return.
	if !metav1.IsControlledBy(ss, cluster) {
		msg := fmt.Sprintf(MessageResourceExists, "StatefulSet", ss.Namespace, ss.Name)
		m.recorder.Event(cluster, corev1.EventTypeWarning, ErrResourceExists, msg)
		return fmt.Errorf(msg)
	}

	// Upgrade the required component resources the current MySQLOperator version.
	if err := m.ensureMySQLOperatorVersion(cluster, ss, buildversion.GetBuildVersion()); err != nil {
		return errors.Wrap(err, "ensuring MySQL Operator version")
	}

	// Upgrade the MySQL server version if required.
	if err := m.ensureMySQLVersion(cluster, ss); err != nil {
		return errors.Wrap(err, "ensuring MySQL version")
	}

	// If this number of the members on the Cluster does not equal the
	// current desired replicas on the StatefulSet, we should update the
	// StatefulSet resource.
	if cluster.Spec.Members != *ss.Spec.Replicas {
		glog.V(4).Infof("Updating %q: clusterMembers=%d statefulSetReplicas=%d",
			nsName, cluster.Spec.Members, ss.Spec.Replicas)
		old := ss.DeepCopy()
		ss = statefulsets.NewForCluster(cluster, m.opConfig.Images, svc.Name)
		if err := m.statefulSetControl.Patch(old, ss); err != nil {
			// Requeue the item so we can attempt processing again later.
			// This could have been caused by a temporary network failure etc.
			return err
		}
	}

	// Finally, we update the status block of the Cluster resource to
	// reflect the current state of the world.
	err = m.updateClusterStatus(cluster, ss)
	if err != nil {
		return err
	}

	m.recorder.Event(cluster, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
	return nil
}

func getMySQLContainerIndex(containers []corev1.Container) (int, error) {
	for i, c := range containers {
		if c.Name == statefulsets.MySQLServerName {
			return i, nil
		}
	}

	return 0, errors.Errorf("no %q container found", statefulsets.MySQLServerName)
}

// splitImage splits an image into its name and version.
func splitImage(image string) (string, string, error) {
	parts := strings.Split(image, ":")
	if len(parts) < 2 {
		return "", "", errors.Errorf("invalid image %q", image)
	}
	return strings.Join(parts[:len(parts)-1], ""), parts[len(parts)-1], nil
}

func (m *MySQLController) ensureMySQLVersion(c *v1alpha1.Cluster, ss *apps.StatefulSet) error {
	index, err := getMySQLContainerIndex(ss.Spec.Template.Spec.Containers)
	if err != nil {
		return errors.Wrapf(err, "getting MySQL container for StatefulSet %q", ss.Name)
	}
	imageName, actualVersion, err := splitImage(ss.Spec.Template.Spec.Containers[index].Image)
	if err != nil {
		return errors.Wrapf(err, "getting MySQL version for StatefulSet %q", ss.Name)
	}

	actual, err := semver.NewVersion(actualVersion)
	if err != nil {
		return errors.Wrap(err, "parsing StatuefulSet MySQL version")
	}
	expected, err := semver.NewVersion(c.Spec.Version)
	if err != nil {
		return errors.Wrap(err, "parsing Cluster MySQL version")
	}

	switch expected.Compare(*actual) {
	case -1:
		return errors.Errorf("attempted unsupported downgrade from %q to %q", actual, expected)
	case 0:
		return nil
	}

	updated := ss.DeepCopy()
	updated.Spec.Template.Spec.Containers[index].Image = fmt.Sprintf("%s:%s", imageName, c.Spec.Version)
	// NOTE: We do this as previously we defaulted to the OnDelete strategy
	// so clusters created with previous versions would not support upgrades.
	updated.Spec.UpdateStrategy = apps.StatefulSetUpdateStrategy{
		Type: apps.RollingUpdateStatefulSetStrategyType,
	}

	err = m.statefulSetControl.Patch(ss, updated)
	if err != nil {
		return errors.Wrap(err, "patching StatefulSet")
	}

	return nil
}

// ensureMySQLOperatorVersion updates the MySQLOperator resource types that
//require it to make it consistent with the specified operator version.
func (m *MySQLController) ensureMySQLOperatorVersion(c *v1alpha1.Cluster, ss *apps.StatefulSet, operatorVersion string) error {
	// Ensure the Pods belonging to the Cluster are updated to the correct 'mysql-agent' image for the current MySQLOperator version.
	container := statefulsets.MySQLAgentName
	pods, err := m.podLister.List(SelectorForCluster(c))
	if err != nil {
		return errors.Wrapf(err, "listing pods matching %q", SelectorForCluster(c).String())
	}
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
		err = m.statefulSetControl.Patch(ss, updated)
		if err != nil {
			return errors.Wrap(err, "upgrade operator version: PatchStatefulSet failed")
		}
	}

	// Ensure the Cluster is updated with the correct MySQLOperator version.
	if !SelectorForClusterOperatorVersion(operatorVersion).Matches(labels.Set(c.Labels)) {
		glog.Infof("Upgrading cluster statefulset '%s/%s' to latest operator version: %s", c.Namespace, c.Name, operatorVersion)
		copy := c.DeepCopy()
		copy.Labels[constants.MySQLOperatorVersionLabel] = operatorVersion
		err := m.clusterUpdater.UpdateClusterLabels(copy, labels.Set(copy.Labels))
		if err != nil {
			return errors.Wrap(err, "upgrade operator version: ClusterUpdate failed")
		}
	}
	return nil
}

// updateClusterStatusForSS updates Cluster statuses based on changes to their associated StatefulSets.
func (m *MySQLController) updateClusterStatus(cluster *v1alpha1.Cluster, ss *apps.StatefulSet) error {
	glog.V(4).Infof("%s/%s: ss.Spec.Replicas=%d, ss.Status.ReadyReplicas=%d, ss.Status.Replicas=%d",
		cluster.Namespace, cluster.Name, *ss.Spec.Replicas, ss.Status.ReadyReplicas, ss.Status.Replicas)

	status := cluster.Status.DeepCopy()
	_, condition := clusterutil.GetClusterCondition(&cluster.Status, v1alpha1.ClusterReady)
	if condition == nil {
		condition = &v1alpha1.ClusterCondition{Type: v1alpha1.ClusterReady}
	}
	if ss.Status.ReadyReplicas == ss.Status.Replicas && ss.Status.ReadyReplicas == cluster.Spec.Members {
		condition.Status = corev1.ConditionTrue
	} else {
		condition.Status = corev1.ConditionFalse
	}

	if updated := clusterutil.UpdateClusterCondition(status, condition); updated {
		return m.clusterUpdater.UpdateClusterStatus(cluster.DeepCopy(), status)
	}
	return nil
}

// enqueueCluster takes a Cluster resource and converts it into a
// namespace/name string which is then put onto the work queue. This method
// should *not* be passed resources of any type other than Cluster.
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
		// If this object is not owned by a Cluster, we should not do
		// anything more with it.
		if ownerRef.Kind != v1alpha1.ClusterCRDResourceKind {
			return
		}

		cluster, err := m.clusterLister.Clusters(object.GetNamespace()).Get(ownerRef.Name)
		if err != nil {
			glog.V(4).Infof("ignoring orphaned object '%s' of Cluster '%s'", object.GetSelfLink(), ownerRef.Name)
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
