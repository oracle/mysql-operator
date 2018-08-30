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

package backup

import (
	"context"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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

	backuputil "github.com/oracle/mysql-operator/pkg/api/backup"
	v1alpha1 "github.com/oracle/mysql-operator/pkg/apis/mysql/v1alpha1"
	clusterlabeler "github.com/oracle/mysql-operator/pkg/controllers/cluster/labeler"
	controllerutils "github.com/oracle/mysql-operator/pkg/controllers/util"
	clientset "github.com/oracle/mysql-operator/pkg/generated/clientset/versioned/typed/mysql/v1alpha1"
	informersv1alpha1 "github.com/oracle/mysql-operator/pkg/generated/informers/externalversions/mysql/v1alpha1"
	listersv1alpha1 "github.com/oracle/mysql-operator/pkg/generated/listers/mysql/v1alpha1"
	kubeutil "github.com/oracle/mysql-operator/pkg/util/kube"
)

const controllerAgentName = "operator-backup-controller"

// OperatorController handles validation, labeling, and scheduling of
// Backups to be executed on a specific (primary) mysql-agent. It is run
// in the operator.
type OperatorController struct {
	client      clientset.BackupsGetter
	syncHandler func(key string) error

	// backupLister is able to list/get Backups from a shared informer's
	// store.
	backupLister listersv1alpha1.BackupLister
	// backupListerSynced returns true if the Backup shared informer has
	// synced at least once.
	backupListerSynced cache.InformerSynced

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

	queue workqueue.RateLimitingInterface
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder

	// conditionUpdater updates the conditions of Backups.
	conditionUpdater ConditionUpdater
}

// NewOperatorController constructs a new OperatorController.
func NewOperatorController(
	kubeClient kubernetes.Interface,
	client clientset.BackupsGetter,
	backupInformer informersv1alpha1.BackupInformer,
	clusterInformer informersv1alpha1.ClusterInformer,
	podInformer corev1informers.PodInformer,
) *OperatorController {
	// Create event broadcaster.
	glog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	c := &OperatorController{
		client:              client,
		backupLister:        backupInformer.Lister(),
		backupListerSynced:  backupInformer.Informer().HasSynced,
		clusterLister:       clusterInformer.Lister(),
		clusterListerSynced: clusterInformer.Informer().HasSynced,
		podLister:           podInformer.Lister(),
		podListerSynced:     podInformer.Informer().HasSynced,
		queue:               workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "backup"),
		recorder:            recorder,
		conditionUpdater:    &conditionUpdater{client: client},
	}

	c.syncHandler = c.processBackup

	backupInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				backup := obj.(*v1alpha1.Backup)

				_, cond := backuputil.GetBackupCondition(&backup.Status, v1alpha1.BackupScheduled)
				if cond != nil && cond.Status == corev1.ConditionTrue {
					glog.V(4).Infof("Backup %q is already scheduled on Cluster member %q",
						kubeutil.NamespaceAndName(backup), backup.Spec.ScheduledMember)
					return
				}

				key, err := cache.MetaNamespaceKeyFunc(backup)
				if err != nil {
					glog.Errorf("Error creating queue key, item not added to queue: %v", err)
					return
				}
				c.queue.Add(key)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				new := newObj.(*v1alpha1.Backup)

				_, cond := backuputil.GetBackupCondition(&new.Status, v1alpha1.BackupComplete)
				if cond != nil && cond.Status == corev1.ConditionTrue {
					glog.V(4).Infof("Backup %q is Complete, skipping.", kubeutil.NamespaceAndName(new))
					return
				}

				_, cond = backuputil.GetBackupCondition(&new.Status, v1alpha1.BackupScheduled)
				if cond != nil && cond.Status == corev1.ConditionTrue {
					glog.V(4).Infof("Backup %q is already scheduled on Cluster member %q",
						kubeutil.NamespaceAndName(new), new.Spec.ScheduledMember)
					return
				}

				key, err := cache.MetaNamespaceKeyFunc(new)
				if err != nil {
					glog.Errorf("Error creating queue key, item not added to queue: %v", err)
					return
				}
				c.queue.Add(key)
				glog.V(4).Infof("Backup %q queued", kubeutil.NamespaceAndName(new))
			},
		},
	)

	return c
}

// Run is a blocking function that runs the specified number of worker
// goroutines to process items in the work queue. It will return when it
// receives on the stopCh channel.
func (controller *OperatorController) Run(ctx context.Context, numWorkers int) error {
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

	glog.Info("Starting OperatorController")
	defer glog.Info("Shutting down OperatorController")

	glog.Info("Waiting for caches to sync")
	if !controllerutils.WaitForCacheSync(controllerAgentName, ctx.Done(),
		controller.backupListerSynced,
		controller.clusterListerSynced,
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

func (controller *OperatorController) runWorker() {
	// Continually take items off the queue (waits if it's empty) until we get a
	// shutdown signal from the queue.
	for controller.processNextWorkItem() {
	}
}

func (controller *OperatorController) processNextWorkItem() bool {
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

func (controller *OperatorController) processBackup(key string) error {
	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return errors.Wrap(err, "error splitting queue key")
	}

	// Get resource from store.
	backup, err := controller.backupLister.Backups(ns).Get(name)
	if err != nil {
		return errors.Wrap(err, "error getting Backup")
	}

	// Don't modify items in the cache.
	backup = backup.DeepCopy()
	// Set defaults (incl. operator version label).
	backup = backup.EnsureDefaults()

	validationErr := backup.Validate()
	if validationErr == nil {
		validationErrs := field.ErrorList{}
		fldPath := field.NewPath("spec")

		// Check the referenced Cluster exists.
		_, err := controller.clusterLister.Clusters(ns).Get(backup.Spec.Cluster.Name)
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return err
			}
			validationErrs = append(validationErrs,
				field.NotFound(fldPath.Child("cluster").Child("name"), backup.Spec.Cluster.Name))
		}

		if len(validationErrs) > 0 {
			validationErr = validationErrs.ToAggregate()
		}
	}

	// TODO(apryde): Maybe we should add an UpdateFunc to the backupInformer and support users fixing
	// validation errors via updates (rather than recreation).
	if validationErr != nil {
		controller.recorder.Eventf(backup, corev1.EventTypeWarning, "FailedValidation", validationErr.Error())
		// NOTE: We only return an error here if we fail to set the condition
		// (rather than on validation failure) as we don't want to retry.
		return controller.conditionUpdater.Update(backup, &v1alpha1.BackupCondition{
			Type:    v1alpha1.BackupFailed,
			Status:  corev1.ConditionFalse,
			Reason:  "FailedValidation",
			Message: validationErr.Error(),
		})
	}

	// If possible schedule backup on a secondary member otherwise a primary.
	backup, err = controller.scheduleBackup(backup)
	if err != nil {
		return errors.Wrap(err, "failed to schedule")
	}

	// Update resource.
	backup, err = controller.client.Backups(ns).Update(backup)
	if err != nil {
		return errors.Wrap(err, "failed to update")
	}

	controller.recorder.Eventf(backup, corev1.EventTypeNormal, "SuccessScheduled", "Scheduled on Pod %q", backup.Spec.ScheduledMember)

	return nil
}

// scheduleBackup schedules a Backup on a specific member of a Cluster.
func (controller *OperatorController) scheduleBackup(backup *v1alpha1.Backup) (*v1alpha1.Backup, error) {
	var (
		name = backup.Spec.Cluster.Name
		ns   = backup.Namespace
	)

	// First try and back up from a secondary.
	secondaries, err := controller.podLister.Pods(ns).List(clusterlabeler.SecondarySelector(name))
	if err != nil {
		return backup, errors.Wrap(err, "error listing Pods to choose secondary")
	}
	if len(secondaries) > 0 {
		backuputil.UpdateBackupCondition(&backup.Status, &v1alpha1.BackupCondition{
			Type:   v1alpha1.BackupScheduled,
			Status: corev1.ConditionTrue,
		})
		backup.Spec.ScheduledMember = secondaries[0].Name
		return backup, nil
	}

	// If no secondaries exist back up on a primary.
	primaries, err := controller.podLister.Pods(ns).List(clusterlabeler.PrimarySelector(name))
	if err != nil {
		return backup, errors.Wrap(err, "error listing Pods to choose primary")
	}
	if len(primaries) > 0 {
		backuputil.UpdateBackupCondition(&backup.Status, &v1alpha1.BackupCondition{
			Type:   v1alpha1.BackupScheduled,
			Status: corev1.ConditionTrue,
		})
		backup.Spec.ScheduledMember = primaries[0].Name
		return backup, nil
	}

	return nil, errors.New("no primaries found")
}
