package backup

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/golang/glog"

	apps "k8s.io/api/apps/v1beta1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	appsinformers "k8s.io/client-go/informers/apps/v1beta1"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes/fake"
	scheme "k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	cache "k8s.io/client-go/tools/cache"
	record "k8s.io/client-go/tools/record"

	api "github.com/oracle/mysql-operator/pkg/apis/mysql/v1"
	"github.com/oracle/mysql-operator/pkg/constants"
	util "github.com/oracle/mysql-operator/pkg/controllers/util"
	mysqlfake "github.com/oracle/mysql-operator/pkg/generated/clientset/versioned/fake"
	mysqlinformerfactory "github.com/oracle/mysql-operator/pkg/generated/informers/externalversions"
	mysqlinformer "github.com/oracle/mysql-operator/pkg/generated/informers/externalversions/mysql/v1"
	statefulsets "github.com/oracle/mysql-operator/pkg/resources/statefulsets"
)

const (
	// The time to wait in seconds for any generated events to propogate.
	EventPropogationTimeout = 3
)

func TestBackupValidationFailure(t *testing.T) {
	name := "test"
	namespace := fmt.Sprintf("%s-ns", name)
	clusterName := fmt.Sprintf("%s-cluster", name)
	backupName := "backup1"
	s3CredsSecretRef := "s3creds"
	databases := []string{"db1", "db2"}
	backup := mockMySQLBackup(namespace, clusterName, backupName, s3CredsSecretRef, databases)
	backup.Spec = api.BackupSpec{} // Invalidate BackupSpec.

	controller, informers := newFakeBackupOperatorController(namespace)
	if controller == nil {
		t.Fatalf("Failed to init fake backup.OperatorController.")
	}
	informers.mySQLClient.MysqlV1().MySQLBackups(backup.Namespace).Create(backup)
	informers.backupInformer.Informer().GetStore().Add(backup)

	enqueueBackup(controller, backup)

	fakeWorker(controller)

	// Check validation error.
	updated, err := informers.mySQLClient.MysqlV1().MySQLBackups(backup.Namespace).Get(backup.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Get client MySQLBackup err: %+v", err)
	}
	if updated.Status.Phase != api.BackupPhaseFailed {
		t.Errorf("Expected MySQLBackup to have status phase of '%s', got '%s'.", api.BackupPhaseFailed, updated.Status.Phase)
	}

	events, err := getEvents(informers.kubeClient.CoreV1().Events(backup.Namespace), EventPropogationTimeout)
	if err != nil {
		t.Fatalf("Get client MySQLBackup Events: %+v", err)
	}
	if len(events.Items) != 1 {
		t.Fatalf("Expected 1 MySQLBackup Events.")
	}
	event := events.Items[0]
	if !strings.HasPrefix(event.Name, backup.Name) {
		t.Errorf("MySQLBackup validation warning event had incorrect name: %s", event.Name)
	}
	if event.Namespace != backup.Namespace {
		t.Errorf("MySQLBackup validation warning event had incorrect namespace: %s", event.Namespace)
	}
	if event.Reason != "FailedValidation" {
		t.Errorf("MySQLBackup validation warning event had incorrect reason: %s", event.Reason)
	}
	if !strings.Contains(event.Message, "spec.executor: Required value: missing executor") {
		t.Errorf("MySQLBackup validation warning event did not contain missing executor clause: %s", event.Message)
	}
	if !strings.Contains(event.Message, "spec.storage: Required value: missing storage") {
		t.Errorf("MySQLBackup validation warning event did not contain missing storage clause: %s", event.Message)
	}
	if !strings.Contains(event.Message, "spec.clusterRef: Required value: missing cluster") {
		t.Errorf("MySQLBackup validation warning event did not contain missing cluster clause: %s", event.Message)
	}
}

func TestBackupNoClusterFailure(t *testing.T) {
	name := "test"
	namespace := fmt.Sprintf("%s-ns", name)
	clusterName := fmt.Sprintf("%s-cluster", name)
	backupName := "backup1"
	s3CredsSecretRef := "s3creds"
	databases := []string{"db1", "db2"}
	backup := mockMySQLBackup(namespace, clusterName, backupName, s3CredsSecretRef, databases)

	controller, informers := newFakeBackupOperatorController(namespace)
	if controller == nil {
		t.Fatalf("Failed to init fake backup.OperatorController.")
	}
	informers.mySQLClient.MysqlV1().MySQLBackups(backup.Namespace).Create(backup)
	informers.backupInformer.Informer().GetStore().Add(backup)

	enqueueBackup(controller, backup)

	fakeWorker(controller)

	// Check errors when cluster is missing.
	updated, err := informers.mySQLClient.MysqlV1().MySQLBackups(backup.Namespace).Get(backup.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Get client MySQLBackup err: %+v", err)
	}
	if updated.Status.Phase != api.BackupPhaseFailed {
		t.Errorf("Expected MySQLBackup to have status phase of '%s', got '%s'.", api.BackupPhaseFailed, updated.Status.Phase)
	}

	events, err := getEvents(informers.kubeClient.CoreV1().Events(backup.Namespace), EventPropogationTimeout)
	if err != nil {
		t.Fatalf("Get client MySQLBackup Events: %+v", err)
	}
	if len(events.Items) != 1 {
		t.Fatalf("Expected 1 MySQLBackup Events.")
	}

	event := events.Items[0]
	if !strings.HasPrefix(event.Name, backup.Name) {
		t.Errorf("MySQLBackup validation warning event had incorrect name: %s", event.Name)
	}
	if event.Namespace != backup.Namespace {
		t.Errorf("MySQLBackup validation warning event had incorrect namespace: %s", event.Namespace)
	}
	if event.Reason != "FailedValidation" {
		t.Errorf("MySQLBackup validation warning event had incorrect reason: %s", event.Reason)
	}
	if !strings.Contains(event.Message, "spec.clusterRef.name: Not found: \"test-cluster\"") {
		t.Errorf("MySQLBackup validation warning event did not reference non-existant cluster: %s", event.Message)
	}
}

func TestScheduleBackupSingleNodeSuccess(t *testing.T) {
	opVersion := "0.0.0"
	name := "test"
	namespace := fmt.Sprintf("%s-ns", name)
	clusterName := fmt.Sprintf("%s-cluster", name)
	replicas := int32(1)
	backupName := "backup1"
	s3CredsSecretRef := "s3creds"
	databases := []string{"db1", "db2"}
	cluster := mockMySQLCluster(opVersion, clusterName, namespace, replicas)
	statefulSet := mockClusterStatefulSet(cluster)
	pods := labelClusterPods(mockClusterPods(statefulSet))
	backup := mockMySQLBackup(namespace, clusterName, backupName, s3CredsSecretRef, databases)

	controller, informers := newFakeBackupOperatorController(namespace)
	if controller == nil {
		t.Fatalf("Failed to init fake backup.OperatorController.")
	}
	informers.mySQLClient.MysqlV1().MySQLClusters(cluster.Namespace).Create(cluster)
	informers.clusterInformer.Informer().GetStore().Add(cluster)
	informers.statefulSetInformer.Informer().GetStore().Add(statefulSet)
	for _, pod := range pods {
		informers.podInformer.Informer().GetStore().Add(pod)
	}
	informers.mySQLClient.MysqlV1().MySQLBackups(backup.Namespace).Create(backup)
	informers.backupInformer.Informer().GetStore().Add(backup)

	enqueueBackup(controller, backup)

	fakeWorker(controller)

	// Check backup is scheduled in single-node cluster. Backup should be
	// scheduled on primary.
	updated, err := informers.mySQLClient.MysqlV1().MySQLBackups(backup.Namespace).Get(backup.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Get client MySQLBackup err: %+v", err)
	}
	if updated.Status.Phase != api.BackupPhaseScheduled {
		t.Errorf("Expected MySQLBackup to have status phase of '%s', got '%s'.", api.BackupPhaseScheduled, updated.Status.Phase)
	}

	// Check the backup scheduled event is generated.
	events, err := getEvents(informers.kubeClient.CoreV1().Events(backup.Namespace), EventPropogationTimeout)
	if err != nil {
		t.Fatalf("Get client MySQLBackup Events: %+v", err)
	}
	if len(events.Items) != 1 {
		t.Fatalf("Expected 1 MySQLBackup Events.")
	}
	event := events.Items[0]
	if !strings.HasPrefix(event.Name, backup.Name) {
		t.Errorf("MySQLBackup SuccessScheduled event had incorrect name: %s", event.Name)
	}
	if event.Namespace != backup.Namespace {
		t.Errorf("MySQLBackup SuccessScheduled event had incorrect namespace: %s", event.Namespace)
	}
	if event.Reason != "SuccessScheduled" {
		t.Errorf("MySQLBackup SuccessScheduled event had incorrect reason: %s", event.Reason)
	}
	if event.Message != "Scheduled on Pod \"test-cluster-0\"" {
		t.Errorf("MySQLBackup SuccessScheduled event did not schedule on primary pod: %s", event.Message)
	}
	if event.Source.Component != "operator-backup-controller" {
		t.Errorf("MySQLBackup SuccessScheduled event did not have correct source component: %s", event.Source.Component)
	}
}

func TestScheduleBackupMultiNodeSuccess(t *testing.T) {
	opVersion := "0.0.0"
	name := "test"
	namespace := fmt.Sprintf("%s-ns", name)
	clusterName := fmt.Sprintf("%s-cluster", name)
	replicas := int32(3)
	backupName := "backup1"
	s3CredsSecretRef := "s3creds"
	databases := []string{"db1", "db2"}
	cluster := mockMySQLCluster(opVersion, clusterName, namespace, replicas)
	statefulSet := mockClusterStatefulSet(cluster)
	pods := labelClusterPods(mockClusterPods(statefulSet))
	backup := mockMySQLBackup(namespace, clusterName, backupName, s3CredsSecretRef, databases)

	controller, informers := newFakeBackupOperatorController(namespace)
	if controller == nil {
		t.Fatalf("Failed to init fake backup.OperatorController.")
	}
	informers.mySQLClient.MysqlV1().MySQLClusters(cluster.Namespace).Create(cluster)
	informers.clusterInformer.Informer().GetStore().Add(cluster)
	informers.statefulSetInformer.Informer().GetStore().Add(statefulSet)
	for _, pod := range pods {
		informers.podInformer.Informer().GetStore().Add(pod)
	}
	informers.mySQLClient.MysqlV1().MySQLBackups(backup.Namespace).Create(backup)
	informers.backupInformer.Informer().GetStore().Add(backup)

	enqueueBackup(controller, backup)

	fakeWorker(controller)

	// Check backup is scheduled in multi-node cluster. Backup should be
	// scheduled on a secondary.
	updated, err := informers.mySQLClient.MysqlV1().MySQLBackups(backup.Namespace).Get(backup.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Get client MySQLBackup err: %+v", err)
	}
	if updated.Status.Phase != api.BackupPhaseScheduled {
		t.Errorf("Expected MySQLBackup to have status phase of '%s', got '%s'.", api.BackupPhaseScheduled, updated.Status.Phase)
	}

	// Check the backup scheduled event is generated.
	events, err := getEvents(informers.kubeClient.CoreV1().Events(backup.Namespace), EventPropogationTimeout)
	if err != nil {
		t.Fatalf("Get client MySQLBackup Events: %+v", err)
	}
	if len(events.Items) != 1 {
		t.Fatalf("Expected 1 MySQLBackup Events.")
	}
	event := events.Items[0]
	if !strings.HasPrefix(event.Name, backup.Name) {
		t.Errorf("MySQLBackup SuccessScheduled event had incorrect name: %s", event.Name)
	}
	if event.Namespace != backup.Namespace {
		t.Errorf("MySQLBackup SuccessScheduled event had incorrect namespace: %s", event.Namespace)
	}
	if event.Reason != "SuccessScheduled" {
		t.Errorf("MySQLBackup SuccessScheduled event had incorrect reason: %s", event.Reason)
	}
	if event.Message == "Scheduled on Pod \"test-cluster-0\"" {
		t.Errorf("MySQLBackup SuccessScheduled event should not be scheduled on primary pod (in multi-node cluster): %s", event.Message)
	}
	if !regexp.MustCompile("Scheduled on Pod \"test-cluster-[12]\"").MatchString(event.Message) {
		t.Errorf("MySQLBackup SuccessScheduled event did not schedule on secondary pod: %s", event.Message)
	}
	if event.Source.Component != "operator-backup-controller" {
		t.Errorf("MySQLBackup SuccessScheduled event did not have correct source component: %s", event.Source.Component)
	}
}

// Get the events associated with a backup. Will re-try until the timeout is
// reached.
func getEvents(eventGetter typedcorev1.EventInterface, timeout time.Duration) (*v1.EventList, error) {
	c := make(chan *v1.EventList, 1)
	go func() {
		for {
			events, err := eventGetter.List(metav1.ListOptions{})
			if err == nil && len(events.Items) > 0 {
				c <- events
			}
			time.Sleep(time.Millisecond * 500)
		}
	}()
	select {
	case events := <-c:
		return events, nil
	case <-time.After(time.Second * timeout):
		return nil, fmt.Errorf("failed to obtain events (timed-out)")
	}
}

// fixtures **********

func mockMySQLBackup(namespace string, clusterName string, backupName string, ossCredsSecretRef string, databases []string) *api.MySQLBackup {
	return &api.MySQLBackup{
		TypeMeta: metav1.TypeMeta{
			Kind:       api.MySQLBackupCRDResourceKind,
			APIVersion: api.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      backupName,
			Namespace: namespace,
			Labels: map[string]string{
				"v1.mysql.oracle.com/version": "0.0.1",
			},
		},
		Spec: api.BackupSpec{
			Executor: &api.Executor{
				Provider:  "mysqldump",
				Databases: databases,
			},
			Storage: &api.Storage{
				Provider: "s3",
				SecretRef: &v1.LocalObjectReference{
					Name: ossCredsSecretRef,
				},
				Config: map[string]string{
					"endpoint": "bristoldev.compat.objectstorage.us-phoenix-1.oraclecloud.com",
					"region":   "us-phoenix-1",
					"bucket":   "trjl-test",
				},
			},
			ClusterRef: &v1.LocalObjectReference{
				Name: clusterName,
			},
		},
	}
}

func mockMySQLCluster(operatorVersion string, name string, namespace string, replicas int32) *api.MySQLCluster {
	cluster := &api.MySQLCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MySQLCluster",
			APIVersion: "mysql.oracle.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    map[string]string{constants.MySQLClusterLabel: name, constants.MySQLOperatorVersionLabel: operatorVersion},
		},
		Spec: api.MySQLClusterSpec{
			Replicas: replicas,
		},
	}
	cluster.EnsureDefaults()
	return cluster
}

func mockClusterStatefulSet(cluster *api.MySQLCluster) *apps.StatefulSet {
	return statefulsets.NewForCluster(cluster, cluster.Name)
}

func mockClusterPods(ss *apps.StatefulSet) []*v1.Pod {
	pods := []*v1.Pod{}
	replicas := int(*ss.Spec.Replicas)
	for i := 0; i < replicas; i++ {
		pods = append(pods, mockClusterPod(ss, i))
	}
	return pods
}

func labelClusterPods(pods []*v1.Pod) []*v1.Pod {
	for idx, pod := range pods {
		if idx < 1 {
			pod.Labels[constants.LabelMySQLClusterRole] = constants.MySQLClusterRolePrimary
		} else {
			pod.Labels[constants.LabelMySQLClusterRole] = constants.MySQLClusterRoleSecondary
		}
	}
	return pods
}

func mockClusterPod(ss *apps.StatefulSet, ordinal int) *v1.Pod {
	clusterName := ss.Name
	operatorVersion := ss.ObjectMeta.Labels[constants.MySQLOperatorVersionLabel]
	image := fmt.Sprintf("%s-%s", statefulsets.AgentImageName, operatorVersion)

	pod := &v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%d", ss.Name, ordinal),
			Namespace: ss.Namespace,
			Labels:    map[string]string{constants.MySQLClusterLabel: clusterName, constants.MySQLOperatorVersionLabel: operatorVersion},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				v1.Container{Name: statefulsets.MySQLAgentContainerName, Image: image},
			},
		},
	}
	return pod
}

// fake backup operator controller **********

func alwaysReady() bool { return true }

func fakeWorker(oc *OperatorController) {
	if obj, done := oc.queue.Get(); !done {
		oc.syncHandler(obj.(string))
		oc.queue.Done(obj)
	}
}

func enqueueBackup(oc *OperatorController, backup interface{}) error {
	key, err := cache.MetaNamespaceKeyFunc(backup)
	if err != nil {
		glog.Errorf("Error creating queue key, item not added to queue: %v", err)
		return err
	}
	oc.queue.Add(key)
	return nil
}

// fakeMySQLControllerInformers contain references to the set of underlying
// informers associated with a newFakeMySQLController.
type fakeBackupOperatorControllerInformers struct {
	mySQLClient         *mysqlfake.Clientset
	kubeClient          *fake.Clientset
	clusterInformer     mysqlinformer.MySQLClusterInformer
	podInformer         coreinformers.PodInformer
	statefulSetInformer appsinformers.StatefulSetInformer
	backupInformer      mysqlinformer.MySQLBackupInformer
}

func newFakeBackupOperatorController(namespace string) (*OperatorController, *fakeBackupOperatorControllerInformers) {
	fakeMySQLClient := mysqlfake.NewSimpleClientset()
	informerFactory := mysqlinformerfactory.NewSharedInformerFactory(fakeMySQLClient, util.NoResyncPeriodFunc())
	clusterInformer := informerFactory.Mysql().V1().MySQLClusters()
	backupInformer := informerFactory.Mysql().V1().MySQLBackups()

	fakeKubeClient := fake.NewSimpleClientset()
	kubeInformerFactory := informers.NewSharedInformerFactory(fakeKubeClient, util.NoResyncPeriodFunc())
	podInformer := kubeInformerFactory.Core().V1().Pods()
	statefulSetInformer := kubeInformerFactory.Apps().V1beta1().StatefulSets()

	fakeController := NewOperatorController(fakeKubeClient, fakeMySQLClient.Mysql(), backupInformer, clusterInformer, podInformer)
	fakeController.backupListerSynced = alwaysReady
	fakeController.clusterListerSynced = alwaysReady
	fakeController.podListerSynced = alwaysReady

	fakeInformers := &fakeBackupOperatorControllerInformers{
		mySQLClient:         fakeMySQLClient,
		kubeClient:          fakeKubeClient,
		clusterInformer:     clusterInformer,
		statefulSetInformer: statefulSetInformer,
		podInformer:         podInformer,
		backupInformer:      backupInformer,
	}

	// TODO: Are we are cheating here by re-creating this event broadcaster
	// with the non-all namespace.  We need to use the provided recorder and
	// get the TestBackupValidationFailure test working.
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: fakeKubeClient.CoreV1().Events(namespace)})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: controllerAgentName})
	fakeController.recorder = recorder

	return fakeController, fakeInformers
}
