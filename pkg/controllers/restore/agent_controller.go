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

package restore

import (
	"context"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	field "k8s.io/apimachinery/pkg/util/validation/field"
	wait "k8s.io/apimachinery/pkg/util/wait"
	corev1informers "k8s.io/client-go/informers/core/v1"
	kubernetes "k8s.io/client-go/kubernetes"
	scheme "k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	corev1listers "k8s.io/client-go/listers/core/v1"
	cache "k8s.io/client-go/tools/cache"
	record "k8s.io/client-go/tools/record"
	workqueue "k8s.io/client-go/util/workqueue"

	restoreutil "github.com/oracle/mysql-operator/pkg/api/restore"
	v1alpha1 "github.com/oracle/mysql-operator/pkg/apis/mysql/v1alpha1"
	backuputil "github.com/oracle/mysql-operator/pkg/backup"
	executor "github.com/oracle/mysql-operator/pkg/backup/executor"
	controllerutils "github.com/oracle/mysql-operator/pkg/controllers/util"
	clientset "github.com/oracle/mysql-operator/pkg/generated/clientset/versioned/typed/mysql/v1alpha1"
	informersv1alpha1 "github.com/oracle/mysql-operator/pkg/generated/informers/externalversions/mysql/v1alpha1"
	listersv1alpha1 "github.com/oracle/mysql-operator/pkg/generated/listers/mysql/v1alpha1"
	kubeutil "github.com/oracle/mysql-operator/pkg/util/kube"
	metrics "github.com/oracle/mysql-operator/pkg/util/metrics"
)

const agentControllerAgentName = "agent-restore-controller"

// AgentController handles the excution of Restores on a specific (primary)
// mysql-agent. It is run in each mysql-agent.
// TODO: Probably should only be run iff the local mysql instance is a primary.
type AgentController struct {
	// podName is the name of the pod the controller is running in.
	podName string

	kubeClient  kubernetes.Interface
	client      clientset.RestoresGetter
	syncHandler func(key string) error

	// restoreLister is able to list/get Restores from a shared informer's
	// store.
	restoreLister listersv1alpha1.RestoreLister
	// restoreListerSynced returns true if the Restore shared informer has
	// synced at least once.
	restoreListerSynced cache.InformerSynced

	// podLister is able to list/get Pods from a shared informer's store.
	podLister corev1listers.PodLister
	// podListerSynced returns true if the Pod shared informer has synced at
	// least once.
	podListerSynced cache.InformerSynced

	// clusterLister is able to list/get Clusters from a shared informer's
	// store.
	clusterLister listersv1alpha1.ClusterLister
	// clusterListerSynced returns true if the Cluster shared informer has
	// synced at least once.
	clusterListerSynced cache.InformerSynced

	// backupLister is able to list/get Backups from a shared informer's
	// store.
	backupLister listersv1alpha1.BackupLister
	// backupListerSynced returns true if the Backup shared informer has
	// synced at least once.
	backupListerSynced cache.InformerSynced

	queue workqueue.RateLimitingInterface
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder

	// conditionUpdater updates the conditions of Backups.
	conditionUpdater ConditionUpdater
}

// NewAgentController constructs a new AgentController.
func NewAgentController(
	kubeClient kubernetes.Interface,
	client clientset.RestoresGetter,
	restoreInformer informersv1alpha1.RestoreInformer,
	clusterInformer informersv1alpha1.ClusterInformer,
	backupInformer informersv1alpha1.BackupInformer,
	podInformer corev1informers.PodInformer,
	podName string,
) *AgentController {
	// Create event broadcaster.
	glog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: agentControllerAgentName})

	c := &AgentController{
		podName:             podName,
		kubeClient:          kubeClient,
		client:              client,
		restoreLister:       restoreInformer.Lister(),
		restoreListerSynced: restoreInformer.Informer().HasSynced,
		clusterLister:       clusterInformer.Lister(),
		clusterListerSynced: clusterInformer.Informer().HasSynced,
		backupLister:        backupInformer.Lister(),
		backupListerSynced:  backupInformer.Informer().HasSynced,
		podLister:           podInformer.Lister(),
		podListerSynced:     podInformer.Informer().HasSynced,
		queue:               workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "restore"),
		recorder:            recorder,
		conditionUpdater:    &conditionUpdater{client: client},
	}

	c.syncHandler = c.processRestore

	restoreInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			UpdateFunc: func(oldObj, newObj interface{}) {
				new := newObj.(*v1alpha1.Restore)

				_, cond := restoreutil.GetRestoreCondition(&new.Status, v1alpha1.RestoreComplete)
				if cond != nil && cond.Status == corev1.ConditionTrue {
					glog.V(2).Infof("Restore %q is Complete, skipping.", kubeutil.NamespaceAndName(new))
					return
				}

				_, cond = restoreutil.GetRestoreCondition(&new.Status, v1alpha1.RestoreRunning)
				if cond != nil && cond.Status == corev1.ConditionTrue {
					glog.V(2).Infof("Restore %q is Running, skipping.", kubeutil.NamespaceAndName(new))
					return
				}

				_, cond = restoreutil.GetRestoreCondition(&new.Status, v1alpha1.RestoreScheduled)
				if cond != nil && cond.Status == corev1.ConditionTrue && new.Spec.ScheduledMember == c.podName {
					key, err := cache.MetaNamespaceKeyFunc(new)
					if err != nil {
						glog.Errorf("Error creating queue key, item not added to queue: %v", err)
						return
					}
					c.queue.Add(key)
					return
				}

				glog.V(4).Infof("Restore %q is not Scheduled on this agent")
			},
		},
	)

	return c
}

// Run is a blocking function that runs the specified number of worker
// goroutines to process items in the work queue. It will return when it
// receives on the stopCh channel.
func (controller *AgentController) Run(ctx context.Context, numWorkers int) error {
	var wg sync.WaitGroup

	defer func() {
		glog.Info("Waiting for workers to finish their work")

		controller.queue.ShutDown()

		// We have to wait here in the deferred function instead of at the
		// bottom of the function body because we have to shut down the queue
		// in order for the workers to shut down gracefully, and we want to shut
		// down the queue via defer and not at the end of the body.
		wg.Wait()

		glog.Info("All workers have finished")

	}()

	glog.Info("Starting AgentController")
	defer glog.Info("Shutting down AgentController")

	glog.Info("Waiting for caches to sync")
	if !controllerutils.WaitForCacheSync(controllerAgentName, ctx.Done(),
		controller.restoreListerSynced,
		controller.clusterListerSynced,
		controller.backupListerSynced,
		controller.podListerSynced) {
		return errors.New("timed out waiting for caches to sync")
	}
	glog.Info("Caches are synced")

	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go func() {
			wait.Until(controller.runWorker, time.Second, ctx.Done())
			wg.Done()
		}()
	}

	<-ctx.Done()

	return nil
}

func (controller *AgentController) runWorker() {
	// Continually take items off the queue (waits if it's empty) until we get a
	// shutdown signal from the queue.
	for controller.processNextWorkItem() {
	}
}

func (controller *AgentController) processNextWorkItem() bool {
	key, quit := controller.queue.Get()
	if quit {
		return false
	}
	// Always call done on this item, since if it fails we'll add it back with
	// rate-limiting below.
	defer controller.queue.Done(key)

	err := controller.syncHandler(key.(string))
	if err == nil {
		// If you had no error, tell the queue to stop tracking history for your
		// key. This will reset things like failure counts for per-item rate
		// limiting.
		controller.queue.Forget(key)
		return true
	}

	glog.Errorf("Error in syncHandler, re-adding %q to queue: %+v", key, err)
	// We had an error processing the item so add it back into the queue for
	// re-processing with rate-limiting.
	controller.queue.AddRateLimited(key)

	return true
}

func (controller *AgentController) processRestore(key string) error {
	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return errors.Wrap(err, "error splitting queue key")
	}

	// Get resource from store.
	restore, err := controller.restoreLister.Restores(ns).Get(name)
	if err != nil {
		return errors.Wrap(err, "error getting Restore")
	}

	// Don't modify items in the cache.
	restore = restore.DeepCopy()
	// Set defaults (incl. operator version label).
	restore = restore.EnsureDefaults()

	var (
		backup *v1alpha1.Backup
		creds  *corev1.Secret
	)

	validationErr := restore.Validate()
	if validationErr == nil {
		// If there are no basic validation errors check the referenced
		// resources exist.
		validationErrs := field.ErrorList{}
		fldPath := field.NewPath("spec")

		// Check the referenced Cluster exists.
		_, err := controller.clusterLister.Clusters(ns).Get(restore.Spec.Cluster.Name)
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return err
			}
			validationErrs = append(validationErrs,
				field.NotFound(fldPath.Child("cluster").Child("name"), restore.Spec.Cluster.Name))
		}

		// Check the referenced Backup exists.
		backup, err = controller.backupLister.Backups(ns).Get(restore.Spec.Backup.Name)
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return err
			}
			validationErrs = append(validationErrs,
				field.NotFound(fldPath.Child("backup").Child("name"), restore.Spec.Backup.Name))
		}

		creds, err = controller.kubeClient.CoreV1().Secrets(ns).Get(backup.Spec.StorageProvider.S3.CredentialsSecret.Name, metav1.GetOptions{})
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return errors.Wrap(err, "getting backup credentials secret")
			}
			validationErrs = append(validationErrs,
				field.NotFound(fldPath.Child("backup").Child("name"), backup.Spec.StorageProvider.S3.CredentialsSecret.Name))
		}
		if len(validationErrs) > 0 {
			validationErr = validationErrs.ToAggregate()
		}
	}

	// If the Restore is not valid emit an event to that effect and mark
	// it as failed.
	// TODO(apryde): Maybe we should add an UpdateFunc to the restoreInformer
	// and support users fixing validation errors via updates (rather than
	// recreation).
	if validationErr != nil {
		controller.recorder.Eventf(restore, corev1.EventTypeWarning, "FailedValidation", validationErr.Error())
		// NOTE: We only return an error here if we fail to set the condition
		// (rather than on validation failure) as we don't want to retry.
		return controller.conditionUpdater.Update(restore, &v1alpha1.RestoreCondition{
			Type:    v1alpha1.RestoreFailed,
			Status:  corev1.ConditionFalse,
			Reason:  "FailedValidation",
			Message: validationErr.Error(),
		})
	}

	err = controller.performRestore(restore, backup, creds)
	if err != nil {
		return err
	}

	return nil
}

func (controller *AgentController) performRestore(restore *v1alpha1.Restore, backup *v1alpha1.Backup, creds *corev1.Secret) error {
	started := time.Now()
	if err := controller.conditionUpdater.Update(restore, &v1alpha1.RestoreCondition{
		Type:   v1alpha1.RestoreRunning,
		Status: corev1.ConditionTrue,
	}); err != nil {
		return err
	}

	// TODO: Should backuputil.NewConfiguredRunner accept a map[string][]byte
	// instead?
	credsMap := make(map[string]string, len(creds.Data))
	for k, v := range creds.Data {
		credsMap[k] = string(v)
	}

	runner, err := backuputil.NewConfiguredRunner(backup.Spec.Executor, executor.DefaultCreds(), backup.Spec.StorageProvider, credsMap)
	if err != nil {
		controller.recorder.Event(restore, corev1.EventTypeWarning, "ExecutionFailed", err.Error())
		return controller.conditionUpdater.Update(restore, &v1alpha1.RestoreCondition{
			Type:    v1alpha1.RestoreFailed,
			Status:  corev1.ConditionFalse,
			Reason:  "ExecutionFailed",
			Message: err.Error(),
		})
	}

	err = runner.Restore(backup.Status.Outcome.Location)
	if err != nil {
		controller.recorder.Event(restore, corev1.EventTypeWarning, "ExecutionFailed", err.Error())
		return controller.conditionUpdater.Update(restore, &v1alpha1.RestoreCondition{
			Type:    v1alpha1.RestoreFailed,
			Status:  corev1.ConditionFalse,
			Reason:  "ExecutionFailed",
			Message: err.Error(),
		})
	}

	finished := time.Now()

	restoreutil.UpdateRestoreCondition(&restore.Status, &v1alpha1.RestoreCondition{
		Type:   v1alpha1.RestoreComplete,
		Status: corev1.ConditionTrue,
	})
	restore.Status.TimeStarted = metav1.Time{Time: started}
	restore.Status.TimeCompleted = metav1.Time{Time: finished}
	restore, err = controller.client.Restores(restore.Namespace).Update(restore)
	if err != nil {
		return errors.Wrapf(err, "failed to mark Restore %q as complete", kubeutil.NamespaceAndName(restore))
	}

	metrics.IncEventCounter(clusterRestoreCount)
	glog.Infof("Restore %q succeeded in %v", restore.Name, finished.Sub(started))
	controller.recorder.Event(restore, corev1.EventTypeNormal, "Complete", "Restore complete")

	return nil
}
