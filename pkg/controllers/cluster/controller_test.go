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
	"fmt"
	"testing"
	"time"

	apps "k8s.io/api/apps/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	appsinformers "k8s.io/client-go/informers/apps/v1beta1"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes/fake"
	cache "k8s.io/client-go/tools/cache"

	"github.com/golang/glog"

	options "github.com/oracle/mysql-operator/cmd/mysql-operator/app/options"
	api "github.com/oracle/mysql-operator/pkg/apis/mysql/v1"
	"github.com/oracle/mysql-operator/pkg/constants"
	"github.com/oracle/mysql-operator/pkg/controllers/util"
	mysqlfake "github.com/oracle/mysql-operator/pkg/generated/clientset/versioned/fake"
	mysqlinformer_factory "github.com/oracle/mysql-operator/pkg/generated/informers/externalversions"
	mysqlinformer "github.com/oracle/mysql-operator/pkg/generated/informers/externalversions/mysql/v1"
	"github.com/oracle/mysql-operator/pkg/resources/secrets"
	statefulsets "github.com/oracle/mysql-operator/pkg/resources/statefulsets"
	buildversion "github.com/oracle/mysql-operator/pkg/version"
)

func mockOperatorConfig() options.MySQLOperatorServer {
	opts := options.MySQLOperatorServer{}
	opts.EnsureDefaults()
	return opts
}

func TestMessageResourceExistsFormatString(t *testing.T) {
	ss := statefulsets.NewForCluster(
		&api.MySQLCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-cluster",
				Namespace: "default",
			},
		},
		mockOperatorConfig().Images,
		"test-cluster",
	)
	expected := "StatefulSet default/test-cluster already exists and is not managed by MySQLCluster"
	msg := fmt.Sprintf(MessageResourceExists, "StatefulSet", ss.Namespace, ss.Name)
	if msg != expected {
		t.Errorf("Got %q, expected %q", msg, expected)
	}
}

func TestSyncBadNameSpaceKeyError(t *testing.T) {
	cluster := mockMySQLCluster(buildversion.GetBuildVersion(), "test-cluster", "test-namespace", int32(3))
	fakeController, _ := newFakeMySQLController(cluster)

	key := "a/bad/namespace/key"
	err := fakeController.syncHandler(key)
	if err != nil {
		t.Errorf("SyncHandler should not return an error when a bad namespace key is specified.")
	}
}

// TODO: mysqlcluster 'test-namespace/test-cluster' in work queue no longer exists
func TestSyncClusterNoLongerExistsError(t *testing.T) {
	cluster := mockMySQLCluster(buildversion.GetBuildVersion(), "test-cluster", "test-namespace", int32(3))
	fakeController, fakeInformers := newFakeMySQLController(cluster)
	fakeInformers.clusterInformer.Informer().GetStore().Delete(cluster)
	key, _ := cache.MetaNamespaceKeyFunc(cluster)
	err := fakeController.syncHandler(key)
	if err != nil {
		t.Errorf("SyncHandler should not return an error when the cluster resource no longer exists. %v", err)
	}
}

func TestSyncClusterValidateError(t *testing.T) {
	cluster := mockMySQLCluster(buildversion.GetBuildVersion(), "test-cluster", "test-namespace", int32(3))
	cluster.Status.Phase = "Bad_Phase"
	fakeController, fakeInformers := newFakeMySQLController(cluster)
	fakeInformers.clusterInformer.Informer().GetStore().Add(cluster)
	key, _ := cache.MetaNamespaceKeyFunc(cluster)
	err := fakeController.syncHandler(key)
	if err == nil {
		t.Errorf("SyncHandler should return an error when the cluster resource is invalid.")
	}
	if err.Error() != `status.phase: Invalid value: "Bad_Phase": invalid phase specified` {
		t.Error("SyncHandler should return the correct error when the cluster resource is invalid: ", err)
	}
}

func TestSyncEnsureClusterLabels(t *testing.T) {
	version := buildversion.GetBuildVersion()
	name := "test-cluster"
	namespace := "test-namespace"
	replicas := int32(3)
	cluster := mockMySQLCluster(version, name, namespace, replicas)
	cluster.Labels = nil

	fakeController, fakeInformers := newFakeMySQLController(cluster)
	fakeInformers.clusterInformer.Informer().GetStore().Add(cluster)
	key, _ := cache.MetaNamespaceKeyFunc(cluster)
	err := fakeController.syncHandler(key)
	if err != nil {
		t.Fatalf("Unexpected MySQLCluster syncHandler error: %+v", err)
	}

	assertOperatorClusterInvariants(t, fakeController, namespace, name, version)
}

func assertOperatorClusterInvariants(t *testing.T, controller *MySQLController, namespace string, name string, version string) {
	cluster, err := controller.opClient.MysqlV1().MySQLClusters(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Get client MySQLCluster err: %+v", err)
	}
	clName := cluster.Labels[constants.MySQLClusterLabel]
	if clName != name {
		t.Errorf("Expected MySQLCluster to have name label '%s', got '%s'.", version, clName)
	}
	clVersion := cluster.Labels[constants.MySQLOperatorVersionLabel]
	if clVersion != version {
		t.Errorf("Expected MySQLCluster to have version label '%s', got '%s'.", version, clVersion)
	}
}

func TestSyncEnsureSecret(t *testing.T) {
	version := buildversion.GetBuildVersion()
	name := "test-cluster"
	namespace := "test-namespace"
	replicas := int32(3)
	cluster := mockMySQLCluster(version, name, namespace, replicas)

	fakeController, fakeInformers := newFakeMySQLController(cluster)
	fakeInformers.clusterInformer.Informer().GetStore().Add(cluster)
	key, _ := cache.MetaNamespaceKeyFunc(cluster)
	err := fakeController.syncHandler(key)
	if err != nil {
		t.Fatalf("Unexpected MySQLCluster syncHandler error: %+v", err)
	}

	assertOperatorSecretInvariants(t, fakeController, cluster)
}

func assertOperatorSecretInvariants(t *testing.T, controller *MySQLController, cluster *api.MySQLCluster) {
	secretName := secrets.GetRootPasswordSecretName(cluster)
	secret, err := controller.kubeClient.CoreV1().Secrets(cluster.Namespace).Get(secretName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Get client MySQLCluster secret error: %+v", err)
	}
	if secret == nil {
		t.Fatalf("Expected MySQLCluster to have an associated secret.")
	}
	if secret.Namespace != cluster.Namespace {
		t.Errorf("Expected MySQLCluster secret to have namespace '%s', got '%s'.", secret.Namespace, cluster.Namespace)
	}
	if secret.Name != secretName {
		t.Errorf("Expected MySQLCluster secret to have name '%s', got '%s'.", secret.Name, secretName)
	}
	if secret.Data["password"] == nil {
		t.Fatalf("Expected MySQLCluster secret to have an associated password.")
	}
}

func TestSyncEnsureService(t *testing.T) {
	version := buildversion.GetBuildVersion()
	name := "test-cluster"
	namespace := "test-namespace"
	replicas := int32(3)
	cluster := mockMySQLCluster(version, name, namespace, replicas)

	fakeController, fakeInformers := newFakeMySQLController(cluster)
	fakeInformers.clusterInformer.Informer().GetStore().Add(cluster)
	key, _ := cache.MetaNamespaceKeyFunc(cluster)
	err := fakeController.syncHandler(key)
	if err != nil {
		t.Fatalf("Unexpected MySQLCluster syncHandler error: %+v", err)
	}

	assertOperatorServiceInvariants(t, fakeController, cluster)
}

func assertOperatorServiceInvariants(t *testing.T, controller *MySQLController, cluster *api.MySQLCluster) {
	kubeClient := controller.kubeClient
	service, err := kubeClient.CoreV1().Services(cluster.Namespace).Get(cluster.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Get client MySQLCluster service error: %+v", err)
	}
	if service == nil {
		t.Fatalf("Expected MySQLCluster to have an associated service.")
	}
	if service.Namespace != cluster.Namespace {
		t.Errorf("Expected MySQLCluster service to have namespace '%s', got '%s'.", service.Namespace, cluster.Namespace)
	}
	if service.Name != cluster.Name {
		t.Errorf("Expected MySQLCluster service to have name '%s', got '%s'.", service.Name, cluster.Name)
	}
	if service.OwnerReferences == nil {
		t.Fatalf("Expected MySQLCluster service to have an associated owner reference to the parent MySQLCluster.")
	} else {
		if !hasOwnerReference(service.OwnerReferences, cluster) {
			t.Errorf("Expected MySQLCluster service to have an associated owner reference to the parent MySQLCluster.")
		}
	}
}

func TestSyncEnsureStatefulSet(t *testing.T) {
	version := buildversion.GetBuildVersion()
	name := "test-cluster"
	namespace := "test-namespace"
	replicas := int32(3)
	cluster := mockMySQLCluster(version, name, namespace, replicas)

	fakeController, fakeInformers := newFakeMySQLController(cluster)
	fakeInformers.clusterInformer.Informer().GetStore().Add(cluster)
	key, _ := cache.MetaNamespaceKeyFunc(cluster)
	err := fakeController.syncHandler(key)
	if err != nil {
		t.Fatalf("Unexpected MySQLCluster syncHandler error: %+v", err)
	}

	assertOperatorStatefulSetInvariants(t, fakeController, cluster)
}

func assertOperatorStatefulSetInvariants(t *testing.T, controller *MySQLController, cluster *api.MySQLCluster) {
	kubeClient := controller.kubeClient
	statefulset, err := kubeClient.AppsV1beta1().StatefulSets(cluster.Namespace).Get(cluster.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Get client MySQLCluster statefulset error: %+v", err)
	}
	if statefulset == nil {
		t.Fatalf("Expected MySQLCluster to have an associated statefulset.")
	}
	if statefulset.Namespace != cluster.Namespace {
		t.Errorf("Expected MySQLCluster statefulset to have namespace '%s', got '%s'.", statefulset.Namespace, cluster.Namespace)
	}
	if statefulset.Name != cluster.Name {
		t.Errorf("Expected MySQLCluster statefulset to have name '%s', got '%s'.", statefulset.Name, cluster.Name)
	}
	if statefulset.OwnerReferences == nil {
		t.Fatalf("Expected MySQLCluster statefulset to have an associated owner reference to the parent MySQLCluster.")
	} else {
		if !hasOwnerReference(statefulset.OwnerReferences, cluster) {
			t.Errorf("Expected MySQLCluster statefulset to have an associated owner reference to the parent MySQLCluster.")
		}
	}
	if *statefulset.Spec.Replicas != cluster.Spec.Replicas {
		t.Errorf("Expected MySQLCluster statefulset to have Replicas '%d', got '%d'.", cluster.Spec.Replicas, *statefulset.Spec.Replicas)
	}
	if statefulset.Spec.Template.Spec.Containers == nil || len(statefulset.Spec.Template.Spec.Containers) != 2 {
		t.Fatalf("Expected MySQLCluster to have an associated statefulset with two pod templates.")
	}
	if !hasContainer(statefulset.Spec.Template.Spec.Containers, "mysql") {
		t.Errorf("Expected MySQLCluster statefulset to have template container 'mysql'.")
	}
	if !hasContainer(statefulset.Spec.Template.Spec.Containers, "mysql-agent") {
		t.Errorf("Expected MySQLCluster statefulset to have template container 'mysql-agent'.")
	}
}

func TestEnsureMySQLOperatorVersionWhenNotRequired(t *testing.T) {
	// Create mock resources.
	originalOperatorVersion := "test-12345"
	name := "test-ensure-operator-version"
	namespace := "test-namespace"
	replicas := int32(3)
	cluster := mockMySQLCluster(originalOperatorVersion, name, namespace, replicas)
	statefulSet := mockClusterStatefulSet(cluster)
	pods := mockClusterPods(statefulSet)

	// Create mock mysqloperator controller.
	clusterController, controllerInformers := newFakeMySQLController(cluster, kuberesources(statefulSet, pods)...)

	// Pre-populate informers.
	controllerInformers.clusterInformer.Informer().GetStore().Add(cluster)
	controllerInformers.statefulSetInformer.Informer().GetStore().Add(statefulSet)
	for _, pod := range pods {
		controllerInformers.podInformer.Informer().GetStore().Add(pod)
	}

	// Test mysql operator version is not updated if not required.
	err := clusterController.ensureMySQLOperatorVersion(cluster, statefulSet, originalOperatorVersion)
	if err != nil {
		t.Fatalf("TestMySQLControllerOperatorUpgrade err: %+v", err)
	}
	assertOperatorVersionInvariants(t, clusterController, namespace, name, originalOperatorVersion)
}

func TestEnsureMySQLOperatorVersionWhenRequired(t *testing.T) {
	// Create mock resources.
	originalOperatorVersion := "test-12345"
	updatedOperatorVersion := "test-67890"
	name := "test-ensure-operator-version"
	namespace := "test-namespace"
	replicas := int32(3)
	cluster := mockMySQLCluster(originalOperatorVersion, name, namespace, replicas)
	statefulSet := mockClusterStatefulSet(cluster)
	pods := mockClusterPods(statefulSet)

	// Create mock mysqloperator controller.
	clusterController, controllerInformers := newFakeMySQLController(cluster, kuberesources(statefulSet, pods)...)

	// Pre-populate informers.
	controllerInformers.clusterInformer.Informer().GetStore().Add(cluster)
	controllerInformers.statefulSetInformer.Informer().GetStore().Add(statefulSet)
	for _, pod := range pods {
		controllerInformers.podInformer.Informer().GetStore().Add(pod)
	}

	// Test mysql operator version is updated if required.
	err := clusterController.ensureMySQLOperatorVersion(cluster, statefulSet, updatedOperatorVersion)
	if err != nil {
		t.Fatalf("TestMySQLControllerOperatorUpgrade err: %+v", err)
	}
	assertOperatorVersionInvariants(t, clusterController, namespace, name, updatedOperatorVersion)
}

// test support functions
func assertOperatorVersionInvariants(t *testing.T, controller *MySQLController, namespace string, name string, version string) {
	expectedImageVersion := mockOperatorConfig().Images.MySQLAgentImage + ":" + version

	// Check MySQLCluster has the correct operator version
	updatedCluster, err := controller.opClient.MysqlV1().MySQLClusters(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Get client MySQLCluster err: %+v", err)
	}
	if !SelectorForClusterOperatorVersion(version).Matches(labels.Set(updatedCluster.Labels)) {
		clVersion := updatedCluster.Labels[constants.MySQLOperatorVersionLabel]
		t.Errorf("Expected MySQLCluster to have version label '%s', got '%s'.", version, clVersion)
	}

	// Check StatefulSets has the correct operator version.
	updatedStatefulSet, err := controller.kubeClient.AppsV1beta1().StatefulSets(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Get client StatefulSet err: %+v", err)
	}
	if !SelectorForClusterOperatorVersion(version).Matches(labels.Set(updatedStatefulSet.Labels)) {
		ssVersion := updatedStatefulSet.ObjectMeta.Labels[constants.MySQLOperatorVersionLabel]
		t.Errorf("Expected StatefulSet to have version label '%s', got '%s'.", version, ssVersion)
	}
	agentContainerName := statefulsets.MySQLAgentName
	for _, container := range updatedStatefulSet.Spec.Template.Spec.Containers {
		if container.Name == agentContainerName {
			updatedImageVersion := container.Image
			if expectedImageVersion != updatedImageVersion {
				t.Errorf("Expected StatefulSet pod to have template image '%s', got '%s'.", expectedImageVersion, updatedImageVersion)
			}
			break
		}
	}

	// Check Pods has the correct operator version.
	updatedPodList, err := controller.kubeClient.CoreV1().Pods(namespace).List(metav1.ListOptions{})
	if err != nil {
		t.Fatalf("Get client PodList err: %+v", err)
	}
	for _, updatedPod := range updatedPodList.Items {
		if !SelectorForClusterOperatorVersion(version).Matches(labels.Set(updatedPod.Labels)) {
			podVersion := updatedPod.ObjectMeta.Labels[constants.MySQLOperatorVersionLabel]
			t.Errorf("Expected Pod to have version label '%s', got '%s'.", version, podVersion)
		}
		agentContainerName := statefulsets.MySQLAgentName
		for _, container := range updatedPod.Spec.Containers {
			if container.Name == agentContainerName {
				updatedImageVersion := container.Image
				if expectedImageVersion != updatedImageVersion {
					t.Errorf("Expected Pod to have image '%s', got '%s'.", expectedImageVersion, updatedImageVersion)
				}
				break
			}
		}
	}
}

func TestMySQLControllerSyncClusterFromScratch(t *testing.T) {
	version := buildversion.GetBuildVersion()
	name := "test-from-scratch-mysql-cluster"
	namespace := "test-namespace"
	replicas := int32(3)
	cluster := mockMySQLCluster(version, name, namespace, replicas)

	// create mock mysqloperator controller and prepoulate infromer
	fakeController, fakeInformers := newFakeMySQLController(cluster)
	fakeInformers.clusterInformer.Informer().GetStore().Add(cluster)

	// Enqueue and then crank the work queue to invoke the controller 'syncHandler' method.
	fakeController.enqueueCluster(cluster)
	fakeWorker(fakeController)
	assertOperatorClusterInvariants(t, fakeController, namespace, name, version)
	assertOperatorSecretInvariants(t, fakeController, cluster)
	assertOperatorServiceInvariants(t, fakeController, cluster)
	assertOperatorStatefulSetInvariants(t, fakeController, cluster)
	assertOperatorVersionInvariants(t, fakeController, namespace, name, version)
	cluster, err := fakeController.opClient.MysqlV1().MySQLClusters(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Get client MySQLCluster err: %+v", err)
	}
}

func hasOwnerReference(ownerReferences []metav1.OwnerReference, cluster *api.MySQLCluster) bool {
	for _, or := range ownerReferences {
		if or.APIVersion == cluster.APIVersion && or.Kind == cluster.Kind && or.Name == cluster.Name {
			return true
		}
	}
	return false
}

func hasContainer(containers []v1.Container, name string) bool {
	for _, container := range containers {
		if container.Name == name {
			return true
		}
	}
	return false
}

// mock objects **********

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
	return statefulsets.NewForCluster(cluster, mockOperatorConfig().Images, cluster.Name)
}

func mockClusterPods(ss *apps.StatefulSet) []*v1.Pod {
	pods := []*v1.Pod{}
	replicas := int(*ss.Spec.Replicas)
	for i := 0; i < replicas; i++ {
		pod := mockClusterPod(ss, i)
		pods = append(pods, pod)
	}
	return pods
}

func mockClusterPod(ss *apps.StatefulSet, ordinal int) *v1.Pod {
	clusterName := ss.Name
	operatorVersion := ss.ObjectMeta.Labels[constants.MySQLOperatorVersionLabel]
	image := fmt.Sprintf("%s-%s", mockOperatorConfig().Images.MySQLAgentImage, operatorVersion)

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
				v1.Container{Name: statefulsets.MySQLAgentName, Image: image},
			},
		},
	}
	return pod
}

// mock MySQLCluster controller **********

func alwaysReady() bool { return true }

// fakeMySQLControllerInformers contain references to the set of underlying informers associated
// with a newFakeMySQLController.
type fakeMySQLControllerInformers struct {
	clusterInformer     mysqlinformer.MySQLClusterInformer
	statefulSetInformer appsinformers.StatefulSetInformer
	podInformer         coreinformers.PodInformer
	serviceInformer     coreinformers.ServiceInformer
	configMapInformer   coreinformers.ConfigMapInformer
}

// newFakeMySQLController creates a new fake MySQLController with a fake mysqlop and kube clients and informers
// for unit testing.
func newFakeMySQLController(cluster *api.MySQLCluster, kuberesources ...runtime.Object) (*MySQLController, *fakeMySQLControllerInformers) {
	mysqlopClient := mysqlfake.NewSimpleClientset(cluster)
	kubeClient := fake.NewSimpleClientset(kuberesources...)

	serverVersion, err := kubeClient.Discovery().ServerVersion()
	if err != nil {
		glog.Fatalf("Failed to discover Kubernetes API server version: %+v", err)
	}

	kubeInformerFactory := informers.NewSharedInformerFactory(kubeClient, util.NoResyncPeriodFunc())
	mysqlopInformerFactory := mysqlinformer_factory.NewSharedInformerFactory(mysqlopClient, util.NoResyncPeriodFunc())

	fakeInformers := &fakeMySQLControllerInformers{
		clusterInformer:     mysqlopInformerFactory.Mysql().V1().MySQLClusters(),
		statefulSetInformer: kubeInformerFactory.Apps().V1beta1().StatefulSets(),
		podInformer:         kubeInformerFactory.Core().V1().Pods(),
		serviceInformer:     kubeInformerFactory.Core().V1().Services(),
		configMapInformer:   kubeInformerFactory.Core().V1().ConfigMaps(),
	}

	fakeController := NewController(
		mockOperatorConfig(),
		mysqlopClient,
		kubeClient,
		serverVersion,
		fakeInformers.clusterInformer,
		fakeInformers.statefulSetInformer,
		fakeInformers.podInformer,
		fakeInformers.serviceInformer,
		fakeInformers.configMapInformer,
		30*time.Second,
		cluster.Namespace)

	fakeController.clusterListerSynced = alwaysReady
	fakeController.statefulSetListerSynced = alwaysReady
	fakeController.podListerSynced = alwaysReady
	fakeController.serviceListerSynced = alwaysReady
	fakeController.configMapListerSynced = alwaysReady

	// Override default control structs with customer fakes.
	fakeController.statefulSetControl = NewFakeStatefulSetControl(fakeController.statefulSetControl, kubeClient)
	fakeController.podControl = NewFakePodControl(fakeController.podControl, kubeClient)

	return fakeController, fakeInformers
}

func kuberesources(ss *apps.StatefulSet, pods []*v1.Pod) []runtime.Object {
	objs := []runtime.Object{}
	objs = append(objs, ss)
	for _, pod := range pods {
		objs = append(objs, pod)
	}
	return objs
}

// fakeWorker manually 'cranks' the fake MySQLController's worker queue.
func fakeWorker(fmsc *MySQLController) {
	if obj, done := fmsc.queue.Get(); !done {
		fmsc.syncHandler(obj.(string))
		fmsc.queue.Done(obj)
	}
}
