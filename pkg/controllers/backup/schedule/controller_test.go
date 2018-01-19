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
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/client-go/kubernetes/fake"
	core "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	"github.com/robfig/cron"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/oracle/mysql-operator/pkg/apis/mysql/v1"
	constants "github.com/oracle/mysql-operator/pkg/constants"
	"github.com/oracle/mysql-operator/pkg/controllers/util"
	mysqlfake "github.com/oracle/mysql-operator/pkg/generated/clientset/versioned/fake"
	informers "github.com/oracle/mysql-operator/pkg/generated/informers/externalversions"
	. "github.com/oracle/mysql-operator/pkg/util/test"
	"github.com/oracle/mysql-operator/pkg/version"
)

const maxNumEventsPerTest = 10

func TestProcessSchedule(t *testing.T) {
	mysqlOperatorVersion := version.GetBuildVersion()

	tests := []struct {
		name                             string
		scheduleKey                      string
		schedule                         *api.MySQLBackupSchedule
		fakeClockTime                    string
		expectedErr                      bool
		expectedSchedulePhaseUpdate      *api.MySQLBackupSchedule
		expectedScheduleLastBackupUpdate *api.MySQLBackupSchedule
		expectedBackupCreate             *api.MySQLBackup
		expectedEvents                   []string
	}{
		{
			name:           "invalid key returns error",
			scheduleKey:    "invalid/key/value",
			expectedErr:    true,
			expectedEvents: []string{},
		},
		{
			name:           "missing schedule returns early without an error",
			scheduleKey:    "foo/bar",
			expectedErr:    false,
			expectedEvents: []string{},
		},
		{
			name:           "schedule with phase FailedValidation does not get processed",
			schedule:       NewTestMySQLBackupSchedule("ns", "name").WithPhase(api.BackupSchedulePhaseFailedValidation).MySQLBackupSchedule,
			expectedErr:    false,
			expectedEvents: []string{},
		},
		{
			name:        "schedule with phase New gets validated and failed if invalid",
			schedule:    NewTestMySQLBackupSchedule("ns", "name").WithPhase(api.BackupSchedulePhaseNew).MySQLBackupSchedule,
			expectedErr: false,
			expectedSchedulePhaseUpdate: NewTestMySQLBackupSchedule("ns", "name").
				WithLabel(constants.MySQLOperatorVersionLabel, mysqlOperatorVersion).
				WithPhase(api.BackupSchedulePhaseFailedValidation).
				MySQLBackupSchedule,
			expectedEvents: []string{"Warning CronScheduleValidationError Schedule must be a non-empty valid Cron expression"},
		},
		{
			name:        "schedule with phase <blank> gets validated and failed if invalid",
			schedule:    NewTestMySQLBackupSchedule("ns", "name").MySQLBackupSchedule,
			expectedErr: false,
			expectedSchedulePhaseUpdate: NewTestMySQLBackupSchedule("ns", "name").
				WithLabel(constants.MySQLOperatorVersionLabel, mysqlOperatorVersion).
				WithPhase(api.BackupSchedulePhaseFailedValidation).
				MySQLBackupSchedule,
			expectedEvents: []string{"Warning CronScheduleValidationError Schedule must be a non-empty valid Cron expression"},
		},
		{
			name:        "schedule with phase Enabled gets re-validated and failed if invalid",
			schedule:    NewTestMySQLBackupSchedule("ns", "name").WithPhase(api.BackupSchedulePhaseEnabled).MySQLBackupSchedule,
			expectedErr: false,
			expectedSchedulePhaseUpdate: NewTestMySQLBackupSchedule("ns", "name").
				WithLabel(constants.MySQLOperatorVersionLabel, mysqlOperatorVersion).
				WithPhase(api.BackupSchedulePhaseFailedValidation).
				MySQLBackupSchedule,
			expectedEvents: []string{"Warning CronScheduleValidationError Schedule must be a non-empty valid Cron expression"},
		},
		{
			name:          "schedule with phase New gets validated and triggers a backup",
			schedule:      NewTestMySQLBackupSchedule("ns", "name").WithPhase(api.BackupSchedulePhaseNew).WithCronSchedule("@every 5m").MySQLBackupSchedule,
			fakeClockTime: "2017-01-01 12:00:00",
			expectedErr:   false,
			expectedSchedulePhaseUpdate: NewTestMySQLBackupSchedule("ns", "name").WithLabel(constants.MySQLOperatorVersionLabel, mysqlOperatorVersion).
				WithPhase(api.BackupSchedulePhaseEnabled).WithCronSchedule("@every 5m").MySQLBackupSchedule,
			expectedBackupCreate: NewTestMySQLBackup().WithNamespace("ns").WithName("name-20170101120000").WithLabel("backup-schedule", "name").MySQLBackup,
			expectedScheduleLastBackupUpdate: NewTestMySQLBackupSchedule("ns", "name").WithLabel(constants.MySQLOperatorVersionLabel, mysqlOperatorVersion).
				WithPhase(api.BackupSchedulePhaseEnabled).WithCronSchedule("@every 5m").WithLastBackupTime("2017-01-01 12:00:00").MySQLBackupSchedule,
			expectedEvents: []string{},
		},
		{
			name: "schedule with phase Enabled gets re-validated and triggers a backup if valid",
			schedule: NewTestMySQLBackupSchedule("ns", "name").WithLabel(constants.MySQLOperatorVersionLabel, mysqlOperatorVersion).
				WithPhase(api.BackupSchedulePhaseEnabled).WithCronSchedule("@every 5m").MySQLBackupSchedule,
			fakeClockTime:        "2017-01-01 12:00:00",
			expectedErr:          false,
			expectedBackupCreate: NewTestMySQLBackup().WithNamespace("ns").WithName("name-20170101120000").WithLabel("backup-schedule", "name").MySQLBackup,
			expectedScheduleLastBackupUpdate: NewTestMySQLBackupSchedule("ns", "name").WithLabel(constants.MySQLOperatorVersionLabel, mysqlOperatorVersion).
				WithPhase(api.BackupSchedulePhaseEnabled).WithCronSchedule("@every 5m").WithLastBackupTime("2017-01-01 12:00:00").MySQLBackupSchedule,
			expectedEvents: []string{},
		},
		{
			name: "schedule that's already run gets LastBackup updated",
			schedule: NewTestMySQLBackupSchedule("ns", "name").WithLabel(constants.MySQLOperatorVersionLabel, mysqlOperatorVersion).
				WithPhase(api.BackupSchedulePhaseEnabled).WithCronSchedule("@every 5m").WithLastBackupTime("2000-01-01 00:00:00").MySQLBackupSchedule,
			fakeClockTime:        "2017-01-01 12:00:00",
			expectedErr:          false,
			expectedBackupCreate: NewTestMySQLBackup().WithNamespace("ns").WithName("name-20170101120000").WithLabel("backup-schedule", "name").MySQLBackup,
			expectedScheduleLastBackupUpdate: NewTestMySQLBackupSchedule("ns", "name").WithLabel(constants.MySQLOperatorVersionLabel, mysqlOperatorVersion).
				WithPhase(api.BackupSchedulePhaseEnabled).WithCronSchedule("@every 5m").WithLastBackupTime("2017-01-01 12:00:00").MySQLBackupSchedule,
			expectedEvents: []string{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var (
				mysqlopclient          = mysqlfake.NewSimpleClientset()
				mysqlopInformerFactory = informers.NewSharedInformerFactory(mysqlopclient, util.NoResyncPeriodFunc())
				kubeclient             = fake.NewSimpleClientset()
			)

			c := NewController(
				mysqlopclient,
				kubeclient,
				mysqlopInformerFactory.Mysql().V1().MySQLBackupSchedules(),
				time.Duration(0),
				metav1.NamespaceDefault,
			)

			recorder := record.NewFakeRecorder(maxNumEventsPerTest)
			c.recorder = recorder

			var (
				testTime time.Time
				err      error
			)
			if test.fakeClockTime != "" {
				testTime, err = time.Parse("2006-01-02 15:04:05", test.fakeClockTime)
				require.NoError(t, err, "unable to parse test.fakeClockTime: %v", err)
			}
			c.clock = clock.NewFakeClock(testTime)

			if test.schedule != nil {
				mysqlopInformerFactory.Mysql().V1().MySQLBackupSchedules().Informer().GetStore().Add(test.schedule)

				// this is necessary so the Update() call returns the appropriate object
				mysqlopclient.PrependReactor("update", "mysqlbackupschedules", func(action core.Action) (bool, runtime.Object, error) {
					obj := action.(core.UpdateAction).GetObject()
					// need to deep copy so we can test the schedule state for each call to update
					return true, obj.DeepCopyObject(), nil
				})
			}

			key := test.scheduleKey
			if key == "" && test.schedule != nil {
				key, err = cache.MetaNamespaceKeyFunc(test.schedule)
				require.NoError(t, err, "error getting key from test.schedule: %v", err)
			}

			err = c.processSchedule(key)

			assert.Equal(t, test.expectedErr, err != nil, "got error %v", err)

			expectedActions := make([]core.Action, 0)

			if upd := test.expectedSchedulePhaseUpdate; upd != nil {
				action := core.NewUpdateAction(
					api.SchemeGroupVersion.WithResource("mysqlbackupschedules"),
					upd.Namespace,
					upd)
				expectedActions = append(expectedActions, action)
			}

			if created := test.expectedBackupCreate; created != nil {
				action := core.NewCreateAction(
					api.SchemeGroupVersion.WithResource("mysqlbackups"),
					created.Namespace,
					created)
				expectedActions = append(expectedActions, action)
			}

			if upd := test.expectedScheduleLastBackupUpdate; upd != nil {
				action := core.NewUpdateAction(
					api.SchemeGroupVersion.WithResource("mysqlbackupschedules"),
					upd.Namespace,
					upd)
				expectedActions = append(expectedActions, action)
			}

			assert.Equal(t, expectedActions, mysqlopclient.Actions())

			events := []string{}
			numEvents := len(recorder.Events)
			for i := 0; i < numEvents; i++ {
				event := <-recorder.Events
				events = append(events, event)
			}
			assert.Equal(t, test.expectedEvents, events)
		})
	}
}

func TestGetNextRunTime(t *testing.T) {
	tests := []struct {
		name                      string
		schedule                  *api.MySQLBackupSchedule
		lastRanOffset             string
		expectedDue               bool
		expectedNextRunTimeOffset string
	}{
		{
			name:                      "first run",
			schedule:                  &api.MySQLBackupSchedule{Spec: api.BackupScheduleSpec{Schedule: "@every 5m"}},
			expectedDue:               true,
			expectedNextRunTimeOffset: "5m",
		},
		{
			name:                      "just ran",
			schedule:                  &api.MySQLBackupSchedule{Spec: api.BackupScheduleSpec{Schedule: "@every 5m"}},
			lastRanOffset:             "0s",
			expectedDue:               false,
			expectedNextRunTimeOffset: "5m",
		},
		{
			name:                      "almost but not quite time to run",
			schedule:                  &api.MySQLBackupSchedule{Spec: api.BackupScheduleSpec{Schedule: "@every 5m"}},
			lastRanOffset:             "4m59s",
			expectedDue:               false,
			expectedNextRunTimeOffset: "5m",
		},
		{
			name:                      "time to run again",
			schedule:                  &api.MySQLBackupSchedule{Spec: api.BackupScheduleSpec{Schedule: "@every 5m"}},
			lastRanOffset:             "5m",
			expectedDue:               true,
			expectedNextRunTimeOffset: "5m",
		},
		{
			name:                      "several runs missed",
			schedule:                  &api.MySQLBackupSchedule{Spec: api.BackupScheduleSpec{Schedule: "@every 5m"}},
			lastRanOffset:             "5h",
			expectedDue:               true,
			expectedNextRunTimeOffset: "5m",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cronSchedule, err := cron.Parse(test.schedule.Spec.Schedule)
			require.NoError(t, err, "unable to parse test.schedule.Spec.Schedule: %v", err)

			testClock := clock.NewFakeClock(time.Now())

			if test.lastRanOffset != "" {
				offsetDuration, err := time.ParseDuration(test.lastRanOffset)
				require.NoError(t, err, "unable to parse test.lastRanOffset: %v", err)

				test.schedule.Status.LastBackup = metav1.Time{Time: testClock.Now().Add(-offsetDuration)}
			}

			nextRunTimeOffset, err := time.ParseDuration(test.expectedNextRunTimeOffset)
			if err != nil {
				panic(err)
			}
			expectedNextRunTime := test.schedule.Status.LastBackup.Add(nextRunTimeOffset)

			due, nextRunTime := getNextRunTime(test.schedule, cronSchedule, testClock.Now())

			assert.Equal(t, test.expectedDue, due)
			// ignore diffs of under a second. the cron library does some rounding.
			assert.WithinDuration(t, expectedNextRunTime, nextRunTime, time.Second)
		})
	}
}

func TestParseCronSchedule(t *testing.T) {
	now := time.Date(2017, 8, 10, 12, 27, 0, 0, time.UTC)

	// Start with a Schedule with:
	// - schedule: once a day at 9am
	// - last backup: 2017-08-10 12:27:00 (just happened)
	s := &api.MySQLBackupSchedule{
		Spec: api.BackupScheduleSpec{
			Schedule: "0 9 * * *",
		},
		Status: api.ScheduleStatus{
			LastBackup: metav1.NewTime(now),
		},
	}

	c, errs := parseCronSchedule(s)
	require.Empty(t, errs)

	// make sure we're not due and next backup is tomorrow at 9am
	due, next := getNextRunTime(s, c, now)
	assert.False(t, due)
	assert.Equal(t, time.Date(2017, 8, 11, 9, 0, 0, 0, time.UTC), next)

	// advance the clock a couple of hours and make sure nothing has changed
	now = now.Add(2 * time.Hour)
	due, next = getNextRunTime(s, c, now)
	assert.False(t, due)
	assert.Equal(t, time.Date(2017, 8, 11, 9, 0, 0, 0, time.UTC), next)

	// advance clock to 1 minute after due time, make sure due=true
	now = time.Date(2017, 8, 11, 9, 1, 0, 0, time.UTC)
	due, next = getNextRunTime(s, c, now)
	assert.True(t, due)
	assert.Equal(t, time.Date(2017, 8, 11, 9, 0, 0, 0, time.UTC), next)

	// record backup time
	s.Status.LastBackup = metav1.NewTime(now)

	// advance clock 1 minute, make sure we're not due and next backup is tomorrow at 9am
	now = time.Date(2017, 8, 11, 9, 2, 0, 0, time.UTC)
	due, next = getNextRunTime(s, c, now)
	assert.False(t, due)
	assert.Equal(t, time.Date(2017, 8, 12, 9, 0, 0, 0, time.UTC), next)
}

func TestGetBackup(t *testing.T) {
	tests := []struct {
		name           string
		schedule       *api.MySQLBackupSchedule
		testClockTime  string
		expectedBackup *api.MySQLBackup
	}{
		{
			name: "ensure name is formatted correctly (AM time)",
			schedule: &api.MySQLBackupSchedule{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "foo",
					Name:      "bar",
				},
				Spec: api.BackupScheduleSpec{
					BackupTemplate: api.BackupSpec{},
				},
			},
			testClockTime: "2017-07-25 09:15:00",
			expectedBackup: &api.MySQLBackup{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "foo",
					Name:      "bar-20170725091500",
				},
				Spec: api.BackupSpec{},
			},
		},
		{
			name: "ensure name is formatted correctly (PM time)",
			schedule: &api.MySQLBackupSchedule{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "foo",
					Name:      "bar",
				},
				Spec: api.BackupScheduleSpec{
					BackupTemplate: api.BackupSpec{},
				},
			},
			testClockTime: "2017-07-25 14:15:00",
			expectedBackup: &api.MySQLBackup{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "foo",
					Name:      "bar-20170725141500",
				},
				Spec: api.BackupSpec{},
			},
		},
		{
			name: "ensure schedule backup template is copied",
			schedule: &api.MySQLBackupSchedule{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "foo",
					Name:      "bar",
				},
				Spec: api.BackupScheduleSpec{
					BackupTemplate: api.BackupSpec{
						Executor: &api.Executor{
							Provider:  "mysqldump",
							Databases: []string{"db1", "db2"},
						},
						Storage: &api.Storage{
							Provider: "s3",
							SecretRef: &corev1.LocalObjectReference{
								Name: "backup-storage-creds",
							},
							Config: map[string]string{
								"endpoint": "endpoint",
								"region":   "region",
								"bucket":   "bucket",
							},
						},
						ClusterRef: &corev1.LocalObjectReference{
							Name: "test-cluster",
						},
						AgentScheduled: "hostname-1",
					},
				},
			},
			testClockTime: "2017-07-25 09:15:00",
			expectedBackup: &api.MySQLBackup{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "foo",
					Name:      "bar-20170725091500",
				},
				Spec: api.BackupSpec{
					Executor: &api.Executor{
						Provider:  "mysqldump",
						Databases: []string{"db1", "db2"},
					},
					Storage: &api.Storage{
						Provider: "s3",
						SecretRef: &corev1.LocalObjectReference{
							Name: "backup-storage-creds",
						},
						Config: map[string]string{
							"endpoint": "endpoint",
							"region":   "region",
							"bucket":   "bucket",
						},
					},
					ClusterRef: &corev1.LocalObjectReference{
						Name: "test-cluster",
					},
					AgentScheduled: "hostname-1",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testTime, err := time.Parse("2006-01-02 15:04:05", test.testClockTime)
			require.NoError(t, err, "unable to parse test.testClockTime: %v", err)

			backup := getBackup(test.schedule, clock.NewFakeClock(testTime).Now())

			assert.Equal(t, test.expectedBackup.Namespace, backup.Namespace)
			assert.Equal(t, test.expectedBackup.Name, backup.Name)
			assert.Equal(t, test.expectedBackup.Spec, backup.Spec)
		})
	}
}
