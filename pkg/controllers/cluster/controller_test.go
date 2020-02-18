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

	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	appsinformers "k8s.io/client-go/informers/apps/v1"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes/fake"
	cache "k8s.io/client-go/tools/cache"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/oracle/mysql-operator/pkg/apis/mysql/v1alpha1"
	"github.com/oracle/mysql-operator/pkg/constants"
	"github.com/oracle/mysql-operator/pkg/controllers/util"
	mysqlfake "github.com/oracle/mysql-operator/pkg/generated/clientset/versioned/fake"
	informerfactory "github.com/oracle/mysql-operator/pkg/generated/informers/externalversions"
	informersv1alpha1 "github.com/oracle/mysql-operator/pkg/generated/informers/externalversions/mysql/v1alpha1"
	operatoropts "github.com/oracle/mysql-operator/pkg/options/operator"
	"github.com/oracle/mysql-operator/pkg/resources/secrets"
	statefulsets "github.com/oracle/mysql-operator/pkg/resources/statefulsets"
	buildversion "github.com/oracle/mysql-operator/pkg/version"
)

func TestGetMySQLContainerIndex(t *testing.T) {
	testCases := map[string]struct {
		containers []v1.Container
		index      int
		errors     bool
	}{
		"empty_errors": {
			containers: []v1.Container{},
			errors:     true,
		},
		"mysql_server_only": {
			containers: []v1.Container{{Name: "mysql"}},
			index:      0,
		},
		"mysql_server_and_agent": {
			containers: []v1.Container{{Name: "mysql-agent"}, {Name: "mysql"}},
			index:      1,
		},
		"mysql_agent_only": {
			containers: []v1.Container{{Name: "mysql-agent"}},
			errors:     true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			index, err := getMySQLContainerIndex(tc.containers)
			if tc.errors {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, index, tc.index)
			}
		})
	}
}

func TestSplitImage(t *testing.T) {
	testCases := map[string]struct {
		image   string
		name    string
		version string
		errors  bool
	}{
		"8.0.11": {
			image:   "mysql/mysql-server:8.0.11",
			name:    "mysql/mysql-server",
			version: "8.0.11",
			errors:  false,
		},
		"invalid": {
			image:  "mysql/mysql-server",
			errors: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			name, version, err := splitImage(tc.image)
			if tc.errors {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, name, tc.name)
				assert.Equal(t, version, tc.version)
			}
		})
	}
}

func mockOperatorConfig() operatoropts.MySQLOperatorOpts {
	opts := operatoropts.MySQLOperatorOpts{}
	opts.EnsureDefaults()
	return opts
}

func TestMessageResourceExistsFormatString(t *testing.T) {
	ss := statefulsets.NewForCluster(
		&v1alpha1.Cluster{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Cluster",
				APIVersion: "mysql.oracle.com/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-cluster",
				Namespace: "default",
			},
		},
		mockOperatorConfig().Images,
		"test-cluster",
	)
	expected := "StatefulSet default/test-cluster already exists and is not managed by Cluster"
	msg := fmt.Sprintf(MessageResourceExists, "StatefulSet", ss.Namespace, ss.Name)
	if msg != expected {
		t.Errorf("Got %q, expected %q", msg, expected)
	}
}

func TestSyncBadNameSpaceKeyError(t *testing.T) {
	cluster := mockCluster(buildversion.GetBuildVersion(), "test-cluster", "test-namespace", int32(3))
	fakeController, _ := newFakeMySQLController(cluster)

	key := "a/bad/namespace/key"
	err := fakeController.syncHandler(key)
	if err != nil {
		t.Errorf("SyncHandler should not return an error when a bad namespace key is specified.")
	}
}

// TODO: mysqlcluster 'test-namespace/test-cluster' in work queue no longer exists
func TestSyncClusterNoLongerExistsError(t *testing.T) {
	cluster := mockCluster(buildversion.GetBuildVersion(), "test-cluster", "test-namespace", int32(3))
	fakeController, fakeInformers := newFakeMySQLController(cluster)
	fakeInformers.clusterInformer.Informer().GetStore().Delete(cluster)
	key, _ := cache.MetaNamespaceKeyFunc(cluster)
	err := fakeController.syncHandler(key)
	if err != nil {
		t.Errorf("SyncHandler should not return an error when the cluster resource no longer exists. %v", err)
	}
}

func TestSyncClusterValidateError(t *testing.T) {
	cluster := mockCluster(buildversion.GetBuildVersion(), "test-cluster-with-a-name-greater-than-twenty-eight-chars-long", "test-namespace", int32(3))
	fakeController, fakeInformers := newFakeMySQLController(cluster)
	fakeInformers.clusterInformer.Informer().GetStore().Add(cluster)
	key, _ := cache.MetaNamespaceKeyFunc(cluster)
	err := fakeController.syncHandler(key)
	if err == nil {
		t.Errorf("SyncHandler should return an error when the cluster resource is invalid.")
	}
	if err.Error() != "validating Cluster: metadata.name: Invalid value: \"test-cluster-with-a-name-greater-than-twenty-eight-chars-long\": longer than maximum supported length 28 (see: https://bugs.mysql.com/bug.php?id=90601)" {
		t.Errorf("SyncHandler should return the correct error when the cluster resource is invalid: %q", err)
	}
}

func TestSyncEnsureClusterLabels(t *testing.T) {
	version := buildversion.GetBuildVersion()
	name := "test-cluster"
	namespace := "test-namespace"
	members := int32(3)
	cluster := mockCluster(version, name, namespace, members)
	cluster.Labels = nil

	fakeController, fakeInformers := newFakeMySQLController(cluster)
	fakeInformers.clusterInformer.Informer().GetStore().Add(cluster)
	key, _ := cache.MetaNamespaceKeyFunc(cluster)
	err := fakeController.syncHandler(key)
	if err != nil {
		t.Fatalf("Unexpected Cluster syncHandler error: %+v", err)
	}

	assertOperatorClusterInvariants(t, fakeController, namespace, name, version)
}

func assertOperatorClusterInvariants(t *testing.T, controller *MySQLController, namespace string, name string, version string) {
	cluster, err := controller.opClient.MySQLV1alpha1().Clusters(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Get client Cluster err: %+v", err)
	}
	clName := cluster.Labels[constants.ClusterLabel]
	if clName != name {
		t.Errorf("Expected Cluster to have name label '%s', got '%s'.", version, clName)
	}
	clVersion := cluster.Labels[constants.MySQLOperatorVersionLabel]
	if clVersion != version {
		t.Errorf("Expected Cluster to have version label '%s', got '%s'.", version, clVersion)
	}
}

func TestSyncEnsureSecret(t *testing.T) {
	version := buildversion.GetBuildVersion()
	name := "test-cluster"
	namespace := "test-namespace"
	members := int32(3)
	cluster := mockCluster(version, name, namespace, members)

	fakeController, fakeInformers := newFakeMySQLController(cluster)
	fakeInformers.clusterInformer.Informer().GetStore().Add(cluster)
	key, _ := cache.MetaNamespaceKeyFunc(cluster)
	err := fakeController.syncHandler(key)
	if err != nil {
		t.Fatalf("Unexpected Cluster syncHandler error: %+v", err)
	}

	assertOperatorSecretInvariants(t, fakeController, cluster)
}

func assertOperatorSecretInvariants(t *testing.T, controller *MySQLController, cluster *v1alpha1.Cluster) {
	secretName := secrets.GetRootPasswordSecretName(cluster)
	secret, err := controller.kubeClient.CoreV1().Secrets(cluster.Namespace).Get(secretName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Get client Cluster secret error: %+v", err)
	}
	if secret == nil {
		t.Fatalf("Expected Cluster to have an associated secret.")
	}
	if secret.Namespace != cluster.Namespace {
		t.Errorf("Expected Cluster secret to have namespace '%s', got '%s'.", secret.Namespace, cluster.Namespace)
	}
	if secret.Name != secretName {
		t.Errorf("Expected Cluster secret to have name '%s', got '%s'.", secret.Name, secretName)
	}
	if secret.Data["password"] == nil {
		t.Fatalf("Expected Cluster secret to have an associated password.")
	}
}

func TestSyncEnsureService(t *testing.T) {
	version := buildversion.GetBuildVersion()
	name := "test-cluster"
	namespace := "test-namespace"
	members := int32(3)
	cluster := mockCluster(version, name, namespace, members)

	fakeController, fakeInformers := newFakeMySQLController(cluster)
	fakeInformers.clusterInformer.Informer().GetStore().Add(cluster)
	key, _ := cache.MetaNamespaceKeyFunc(cluster)
	err := fakeController.syncHandler(key)
	if err != nil {
		t.Fatalf("Unexpected Cluster syncHandler error: %+v", err)
	}

	assertOperatorServiceInvariants(t, fakeController, cluster)
}

func assertOperatorServiceInvariants(t *testing.T, controller *MySQLController, cluster *v1alpha1.Cluster) {
	kubeClient := controller.kubeClient
	service, err := kubeClient.CoreV1().Services(cluster.Namespace).Get(cluster.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Get client Cluster service error: %+v", err)
	}
	if service == nil {
		t.Fatalf("Expected Cluster to have an associated service.")
	}
	if service.Namespace != cluster.Namespace {
		t.Errorf("Expected Cluster service to have namespace '%s', got '%s'.", service.Namespace, cluster.Namespace)
	}
	if service.Name != cluster.Name {
		t.Errorf("Expected Cluster service to have name '%s', got '%s'.", service.Name, cluster.Name)
	}
	if service.OwnerReferences == nil {
		t.Fatalf("Expected Cluster service to have an associated owner reference to the parent Cluster.")
	} else {
		if !hasOwnerReference(service.OwnerReferences, cluster) {
			t.Errorf("Expected Cluster service to have an associated owner reference to the parent Cluster.")
		}
	}
}

func TestSyncEnsureStatefulSet(t *testing.T) {
	version := buildversion.GetBuildVersion()
	name := "test-cluster"
	namespace := "test-namespace"
	members := int32(3)
	cluster := mockCluster(version, name, namespace, members)

	fakeController, fakeInformers := newFakeMySQLController(cluster)
	fakeInformers.clusterInformer.Informer().GetStore().Add(cluster)
	key, _ := cache.MetaNamespaceKeyFunc(cluster)
	err := fakeController.syncHandler(key)
	if err != nil {
		t.Fatalf("Unexpected Cluster syncHandler error: %+v", err)
	}

	assertOperatorStatefulSetInvariants(t, fakeController, cluster)
}

func assertOperatorStatefulSetInvariants(t *testing.T, controller *MySQLController, cluster *v1alpha1.Cluster) {
	kubeClient := controller.kubeClient
	statefulset, err := kubeClient.AppsV1().StatefulSets(cluster.Namespace).Get(cluster.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Get client Cluster statefulset error: %+v", err)
	}
	if statefulset == nil {
		t.Fatalf("Expected Cluster to have an associated statefulset.")
	}
	if statefulset.Namespace != cluster.Namespace {
		t.Errorf("Expected Cluster statefulset to have namespace '%s', got '%s'.", statefulset.Namespace, cluster.Namespace)
	}
	if statefulset.Name != cluster.Name {
		t.Errorf("Expected Cluster statefulset to have name '%s', got '%s'.", statefulset.Name, cluster.Name)
	}
	if statefulset.OwnerReferences == nil {
		t.Fatalf("Expected Cluster statefulset to have an associated owner reference to the parent Cluster.")
	} else {
		if !hasOwnerReference(statefulset.OwnerReferences, cluster) {
			t.Errorf("Expected Cluster statefulset to have an associated owner reference to the parent Cluster.")
		}
	}
	if *statefulset.Spec.Replicas != cluster.Spec.Members {
		t.Errorf("Expected Cluster statefulset to have Replicas '%d', got '%d'.", cluster.Spec.Members, *statefulset.Spec.Replicas)
	}
	if statefulset.Spec.Template.Spec.Containers == nil || len(statefulset.Spec.Template.Spec.Containers) != 2 {
		t.Fatalf("Expected Cluster to have an associated statefulset with two pod templates.")
	}
	if !hasContainer(statefulset.Spec.Template.Spec.Containers, "mysql") {
		t.Errorf("Expected Cluster statefulset to have template container 'mysql'.")
	}
	if !hasContainer(statefulset.Spec.Template.Spec.Containers, "mysql-agent") {
		t.Errorf("Expected Cluster statefulset to have template container 'mysql-agent'.")
	}
}

func TestEnsureMySQLOperatorVersionWhenNotRequired(t *testing.T) {
	// Create mock resources.
	originalOperatorVersion := "test-12345"
	name := "test-ensure-operator-version"
	namespace := "test-namespace"
	members := int32(3)
	cluster := mockCluster(originalOperatorVersion, name, namespace, members)
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
	members := int32(3)
	cluster := mockCluster(originalOperatorVersion, name, namespace, members)
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

	// Check Cluster has the correct operator version
	updatedCluster, err := controller.opClient.MySQLV1alpha1().Clusters(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Get client Cluster err: %+v", err)
	}
	if !SelectorForClusterOperatorVersion(version).Matches(labels.Set(updatedCluster.Labels)) {
		clVersion := updatedCluster.Labels[constants.MySQLOperatorVersionLabel]
		t.Errorf("Expected Cluster to have version label '%s', got '%s'.", version, clVersion)
	}

	// Check StatefulSets has the correct operator version.
	updatedStatefulSet, err := controller.kubeClient.AppsV1().StatefulSets(namespace).Get(name, metav1.GetOptions{})
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
	name := "from-scratch"
	namespace := "test-namespace"
	members := int32(3)
	cluster := mockCluster(version, name, namespace, members)

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
	_, err := fakeController.opClient.MySQLV1alpha1().Clusters(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Get client Cluster err: %+v", err)
	}
}

func hasOwnerReference(ownerReferences []metav1.OwnerReference, cluster *v1alpha1.Cluster) bool {
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

func mockCluster(operatorVersion string, name string, namespace string, members int32) *v1alpha1.Cluster {
	cluster := &v1alpha1.Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Cluster",
			APIVersion: "mysql.oracle.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    map[string]string{constants.ClusterLabel: name, constants.MySQLOperatorVersionLabel: operatorVersion},
		},
		Spec: v1alpha1.ClusterSpec{
			Members: members,
		},
	}
	cluster.EnsureDefaults()
	return cluster
}

func mockClusterStatefulSet(cluster *v1alpha1.Cluster) *apps.StatefulSet {
	return statefulsets.NewForCluster(cluster, mockOperatorConfig().Images, cluster.Name)
}

func mockClusterPods(ss *apps.StatefulSet) []*v1.Pod {
	pods := []*v1.Pod{}
	members := int(*ss.Spec.Replicas)
	for i := 0; i < members; i++ {
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
			Labels:    map[string]string{constants.ClusterLabel: clusterName, constants.MySQLOperatorVersionLabel: operatorVersion},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{Name: statefulsets.MySQLAgentName, Image: image},
			},
		},
	}
	return pod
}

// mock Cluster controller **********

func alwaysReady() bool { return true }

// fakeMySQLControllerInformers contain references to the set of underlying informers associated
// with a newFakeMySQLController.
type fakeMySQLControllerInformers struct {
	clusterInformer     informersv1alpha1.ClusterInformer
	statefulSetInformer appsinformers.StatefulSetInformer
	podInformer         coreinformers.PodInformer
	serviceInformer     coreinformers.ServiceInformer
}

// newFakeMySQLController creates a new fake MySQLController with a fake mysqlop and kube clients and informers
// for unit testing.
func newFakeMySQLController(cluster *v1alpha1.Cluster, kuberesources ...runtime.Object) (*MySQLController, *fakeMySQLControllerInformers) {
	mysqlopClient := mysqlfake.NewSimpleClientset()
	// NOTE: Must call Create rather than pass objects to NewSimpleClientset() as the
	// fake client's UnsafeGuessKindToResource maps the kind to clusters rather than
	// mysqlclusters.
	_, err := mysqlopClient.MySQLV1alpha1().Clusters(cluster.Namespace).Create(cluster)
	if err != nil {
		panic(err)
	}
	kubeClient := fake.NewSimpleClientset(kuberesources...)

	kubeInformerFactory := informers.NewSharedInformerFactory(kubeClient, util.NoResyncPeriodFunc())
	mysqlopInformerFactory := informerfactory.NewSharedInformerFactory(mysqlopClient, util.NoResyncPeriodFunc())

	fakeInformers := &fakeMySQLControllerInformers{
		clusterInformer:     mysqlopInformerFactory.MySQL().V1alpha1().Clusters(),
		statefulSetInformer: kubeInformerFactory.Apps().V1().StatefulSets(),
		podInformer:         kubeInformerFactory.Core().V1().Pods(),
		serviceInformer:     kubeInformerFactory.Core().V1().Services(),
	}

	fakeController := NewController(
		mockOperatorConfig(),
		mysqlopClient,
		kubeClient,
		fakeInformers.clusterInformer,
		fakeInformers.statefulSetInformer,
		fakeInformers.podInformer,
		fakeInformers.serviceInformer,
		30*time.Second,
		cluster.Namespace)

	fakeController.clusterListerSynced = alwaysReady
	fakeController.statefulSetListerSynced = alwaysReady
	fakeController.podListerSynced = alwaysReady
	fakeController.serviceListerSynced = alwaysReady

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
