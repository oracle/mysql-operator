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

package backupschedule

import (
	"context"
	"fmt"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	scheme "k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"k8s.io/client-go/util/workqueue"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/robfig/cron"

	api "github.com/oracle/mysql-operator/pkg/apis/mysql/v1"
	mysqlop "github.com/oracle/mysql-operator/pkg/generated/clientset/versioned"
	opinformers "github.com/oracle/mysql-operator/pkg/generated/informers/externalversions/mysql/v1"
	oplisters "github.com/oracle/mysql-operator/pkg/generated/listers/mysql/v1"
)

const controllerName = "backupschedule-controller"

const (
	// CronScheduleValidationError is used as part of the Event 'reason' when a
	// MySQLBackupSchedule fails validation due to an invalid Cron schedule string.
	CronScheduleValidationError = "CronScheduleValidationError"
)

// Controller watches the Kubernetes API for changes to MySQLBackupSchedule
// resources.
type Controller struct {
	opClient                   mysqlop.Interface
	backupScheduleLister       oplisters.MySQLBackupScheduleLister
	backupScheduleListerSynced cache.InformerSynced
	syncHandler                func(scheduleName string) error
	queue                      workqueue.RateLimitingInterface
	syncPeriod                 time.Duration
	clock                      clock.Clock
	namespace                  string
	recorder                   record.EventRecorder
}

// NewController creates a new BackupScheduleController.
func NewController(
	opClient mysqlop.Interface,
	kubeClient kubernetes.Interface,
	backupScheduleInformer opinformers.MySQLBackupScheduleInformer,
	syncPeriod time.Duration,
	namespace string,
) *Controller {
	glog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerName})

	c := &Controller{
		opClient:                   opClient,
		backupScheduleLister:       backupScheduleInformer.Lister(),
		backupScheduleListerSynced: backupScheduleInformer.Informer().HasSynced,
		queue:      workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "backupschedule"),
		syncPeriod: syncPeriod,
		clock:      clock.RealClock{},
		namespace:  namespace,
		recorder:   recorder,
	}

	c.syncHandler = c.processSchedule

	backupScheduleInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				bs := obj.(*api.MySQLBackupSchedule)

				switch bs.Status.Phase {
				case "", api.BackupSchedulePhaseNew, api.BackupSchedulePhaseEnabled:
					// add to work queue
				default:
					glog.V(4).Info("Backup schedule is not new, skipping")
					return
				}

				key, err := cache.MetaNamespaceKeyFunc(bs)
				if err != nil {
					glog.Errorf("Error creating queue key, item not added to queue: %v", err)
					return
				}
				c.queue.Add(key)
			},
		},
	)

	return c
}

// Run is a blocking function that runs the specified number of worker goroutines
// to process items in the work queue.
func (controller *Controller) Run(ctx context.Context, numWorkers int) error {
	var wg sync.WaitGroup

	defer func() {
		glog.V(4).Info("Waiting for workers to finish their work")

		controller.queue.ShutDown()

		// We have to wait here in the deferred function instead of at the bottom of the function body
		// because we have to shut down the queue in order for the workers to shut down gracefully, and
		// we want to shut down the queue via defer and not at the end of the body.
		wg.Wait()

		glog.Info("All workers have finished")
	}()

	glog.V(4).Info("Starting backup schedule controller")
	defer glog.Info("Shutting down backup schedule controller")

	glog.V(2).Info("Waiting for backup schedule controller caches to sync")
	if !cache.WaitForCacheSync(ctx.Done(), controller.backupScheduleListerSynced) {
		return errors.New("timed out waiting for backup schedule controller caches to sync")
	}
	glog.V(2).Info("Backup schedule controller caches are synced")

	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go func() {
			wait.Until(controller.runWorker, time.Second, ctx.Done())
			wg.Done()
		}()
	}

	go wait.Until(controller.enqueueAllEnabledSchedules, controller.syncPeriod, ctx.Done())

	<-ctx.Done()
	return nil
}

func (controller *Controller) enqueueAllEnabledSchedules() {
	backupSchedules, err := controller.backupScheduleLister.MySQLBackupSchedules(controller.namespace).List(labels.NewSelector())
	if err != nil {
		glog.Errorf("Error listing MySQLBackupSchedules: %v", err)
		return
	}

	for _, bs := range backupSchedules {
		if bs.Status.Phase != api.BackupSchedulePhaseEnabled {
			continue
		}

		key, err := cache.MetaNamespaceKeyFunc(bs)
		if err != nil {
			glog.Errorf("Error creating queue key, item not added to queue: %v", err)
			continue
		}
		controller.queue.Add(key)
	}
}

func (controller *Controller) runWorker() {
	// Continually take items off the queue (waits if it's
	// empty) until we get a shutdown signal from the queue
	for controller.processNextWorkItem() {
	}
}

func (controller *Controller) processNextWorkItem() bool {
	key, quit := controller.queue.Get()
	if quit {
		return false
	}
	// Always call done on this item, since if it fails we'll add
	// it back with rate-limiting below
	defer controller.queue.Done(key)

	err := controller.syncHandler(key.(string))
	if err == nil {
		// If you had no error, tell the queue to stop tracking history for your key. This will reset
		// things like failure counts for per-item rate limiting.
		controller.queue.Forget(key)
		return true
	}

	glog.Errorf("Error in syncHandler, re-adding item to queue, key: %v, err: %v", key, err)
	// we had an error processing the item so add it back
	// into the queue for re-processing with rate-limiting
	controller.queue.AddRateLimited(key)

	return true
}

func (controller *Controller) processSchedule(key string) error {
	glog.V(6).Infof("Running processSchedule: key: %s", key)
	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return errors.Wrap(err, "error splitting queue key")
	}

	glog.V(6).Info("Getting backup schedule")
	bs, err := controller.backupScheduleLister.MySQLBackupSchedules(ns).Get(name)
	if err != nil {
		// backup schedule no longer exists
		if apierrors.IsNotFound(err) {
			glog.Errorf("Backup schedule not found, err: %v", err)
			return nil
		}
		return errors.Wrap(err, "error getting MySQLBackupSchedule")
	}

	switch bs.Status.Phase {
	case "", api.BackupSchedulePhaseNew, api.BackupSchedulePhaseEnabled:
		// valid phase for processing
	default:
		return nil
	}

	glog.V(6).Info("Cloning backup schedule")
	// don't modify items in the cache
	bs = bs.DeepCopy().EnsureDefaults()
	err = bs.Validate()
	if err != nil {
		glog.Errorf("Backup schedule validation failed, err: %v", err)
		controller.recorder.Event(bs, corev1.EventTypeWarning, "FailedValidation", err.Error())
		return err
	}

	// validation - even if the item is Enabled, we can't trust it
	// so re-validate
	currentPhase := bs.Status.Phase

	cronSchedule, errs := parseCronSchedule(bs)
	if len(errs) > 0 {
		bs.Status.Phase = api.BackupSchedulePhaseFailedValidation
		for _, err := range errs {
			controller.recorder.Event(bs, corev1.EventTypeWarning, CronScheduleValidationError, err)
		}
	} else {
		bs.Status.Phase = api.BackupSchedulePhaseEnabled
	}

	// update status if it's changed
	if currentPhase != bs.Status.Phase {
		var updatedBackupSchedule *api.MySQLBackupSchedule
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			updatedBackupSchedule, err = controller.opClient.MysqlV1().MySQLBackupSchedules(ns).Update(bs)
			if err != nil {
				return errors.Wrapf(err, "error updating backup schedule phase to %q", bs.Status.Phase)
			}
			return nil
		})
		if err != nil {
			return err
		}
		bs = updatedBackupSchedule
	}

	if bs.Status.Phase != api.BackupSchedulePhaseEnabled {
		return nil
	}

	// check for the backup schedule being due to run, and submit a Backup if so
	return controller.submitBackupIfDue(bs, cronSchedule)
}

func parseCronSchedule(item *api.MySQLBackupSchedule) (cron.Schedule, []string) {
	var validationErrors []string
	var schedule cron.Schedule

	// cron.Parse panics if schedule is empty
	if len(item.Spec.Schedule) == 0 {
		validationErrors = append(validationErrors, "Schedule must be a non-empty valid Cron expression")
		return nil, validationErrors
	}

	// adding a recover() around cron.Parse because it panics on empty string and is possible
	// that it panics under other scenarios as well.
	func() {
		defer func() {
			if r := recover(); r != nil {
				glog.Errorf("Panic parsing schedule: %v, r: %v", item.Spec.Schedule, r)
				validationErrors = append(validationErrors, fmt.Sprintf("invalid schedule: %v", r))
			}
		}()

		if res, err := cron.ParseStandard(item.Spec.Schedule); err != nil {
			glog.Errorf("Error parsing schedule: %v, err: %v", item.Spec.Schedule, err)
			validationErrors = append(validationErrors, fmt.Sprintf("invalid schedule: %v", err))
		} else {
			schedule = res
		}
	}()

	if len(validationErrors) > 0 {
		return nil, validationErrors
	}

	return schedule, nil
}

func (controller *Controller) submitBackupIfDue(item *api.MySQLBackupSchedule, cronSchedule cron.Schedule) error {
	var (
		now                = controller.clock.Now()
		isDue, nextRunTime = getNextRunTime(item, cronSchedule, now)
	)

	if !isDue {
		glog.V(4).Infof("Backup schedule %s[%s] is not due, skipping. nextRunTime: %v", item.Name, item.Spec.Schedule, nextRunTime)
		return nil
	}

	// Don't attempt to "catch up" if there are any missed or failed runs - simply
	// trigger a Backup if it's time.
	glog.Infof("Backup schedule %s[%s] is due, submitting Backup", item.Name, item.Spec.Schedule)
	backup := getBackup(item, now)
	if _, err := controller.opClient.MysqlV1().MySQLBackups(backup.Namespace).Create(backup); err != nil {
		return errors.Wrap(err, "error creating MySQLBackup")
	}

	bs := item.DeepCopy()

	bs.Status.LastBackup = metav1.NewTime(now)

	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if _, err := controller.opClient.MysqlV1().MySQLBackupSchedules(bs.Namespace).Update(bs); err != nil {
			return errors.Wrapf(err, "error updating backup schedule's LastBackup time to %v", bs.Status.LastBackup)
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

// getNextRunTime gets the latest run time (if the backup schedule hasn't run
// yet, this will be the zero value which will trigger an immediate backup).
func getNextRunTime(bs *api.MySQLBackupSchedule, cronSchedule cron.Schedule, asOf time.Time) (bool, time.Time) {
	lastBackupTime := bs.Status.LastBackup.Time

	nextRunTime := cronSchedule.Next(lastBackupTime)

	return asOf.After(nextRunTime), nextRunTime
}

func getBackup(item *api.MySQLBackupSchedule, timestamp time.Time) *api.MySQLBackup {
	backup := &api.MySQLBackup{
		Spec: item.Spec.BackupTemplate,
		ObjectMeta: metav1.ObjectMeta{
			Namespace: item.Namespace,
			Name:      fmt.Sprintf("%s-%s", item.Name, timestamp.Format("20060102150405")),
			Labels: map[string]string{
				"backup-schedule": item.Name,
			},
		},
	}
	return backup
}
