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

package framework

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"

	extensionsv1betav1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	wait "k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	clientcmd "k8s.io/client-go/tools/clientcmd"

	mysqlclientset "github.com/oracle/mysql-operator/pkg/generated/clientset/versioned"
)

const (
	// Poll defines how regularly to poll kubernetes resources.
	Poll = 2 * time.Second
	// DefaultTimeout is how long we wait for long-running operations in the
	// test suite before giving up.
	DefaultTimeout = 10 * time.Minute
)

// Framework is used in the execution of e2e tests.
type Framework struct {
	BaseName          string
	OperatorInstalled bool

	ClientSet      clientset.Interface
	MySQLClientSet mysqlclientset.Interface

	Namespace          *v1.Namespace   // Every test has at least one namespace unless creation is skipped
	namespacesToDelete []*v1.Namespace // Some tests have more than one.

	// To make sure that this framework cleans up after itself, no matter what,
	// we install a Cleanup action before each test and clear it after.  If we
	// should abort, the AfterSuite hook should run all Cleanup actions.
	cleanupHandle CleanupActionHandle

	werckerReportArtifactsDir string
}

// NewDefaultFramework constructs a new e2e test Framework with default options.
func NewDefaultFramework(baseName string) *Framework {
	f := NewFramework(baseName, nil)
	return f
}

// NewFramework constructs a new e2e test Framework.
func NewFramework(baseName string, client clientset.Interface) *Framework {
	f := &Framework{
		BaseName:                  baseName,
		ClientSet:                 client,
		werckerReportArtifactsDir: os.Getenv("WERCKER_REPORT_ARTIFACTS_DIR"),
	}

	BeforeEach(f.BeforeEach)
	AfterEach(f.AfterEach)

	return f
}

// CreateNamespace creates a e2e test namespace.
func (f *Framework) CreateNamespace(baseName string, labels map[string]string) (*v1.Namespace, error) {
	if labels == nil {
		labels = map[string]string{}
	}

	namespaceObj := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("mysql-operator-e2e-tests-%v-", baseName),
			Namespace:    "",
			Labels:       labels,
		},
		Status: v1.NamespaceStatus{},
	}

	// Be robust about making the namespace creation call.
	var got *v1.Namespace
	if err := wait.PollImmediate(Poll, 30*time.Second, func() (bool, error) {
		var err error
		got, err = f.ClientSet.CoreV1().Namespaces().Create(namespaceObj)
		if err != nil {
			Logf("Unexpected error while creating namespace: %v", err)
			return false, nil
		}
		return true, nil
	}); err != nil {
		return nil, err
	}

	if got != nil {
		f.namespacesToDelete = append(f.namespacesToDelete, got)
	}

	return got, nil
}

// DeleteNamespace deletes a given namespace and waits until its contents are
// deleted.
func (f *Framework) DeleteNamespace(namespace string, timeout time.Duration) error {
	startTime := time.Now()
	if err := f.ClientSet.CoreV1().Namespaces().Delete(namespace, nil); err != nil {
		if apierrors.IsNotFound(err) {
			Logf("Namespace %v was already deleted", namespace)
			return nil
		}
		return err
	}

	// wait for namespace to delete or timeout.
	err := wait.PollImmediate(Poll, timeout, func() (bool, error) {
		if _, err := f.ClientSet.CoreV1().Namespaces().Get(namespace, metav1.GetOptions{}); err != nil {
			if apierrors.IsNotFound(err) {
				return true, nil
			}
			Logf("Error while waiting for namespace to be terminated: %v", err)
			return false, nil
		}
		return false, nil
	})

	// Namespace deletion timed out.
	if err != nil {
		return fmt.Errorf("namespace %v was not deleted with limit: %v", namespace, err)
	}

	Logf("namespace %v deletion completed in %s", namespace, time.Now().Sub(startTime))
	return nil
}

// InstallOperator installs the MySQL operator into the given namespace via helm.
// NOTE: Requires that the MySQL operator CRDs have already been installed.
func (f *Framework) InstallOperator(namespace string) error {
	By(fmt.Sprintf("Installing the operator via helm into namespace %q", namespace))
	// TODO(apryde): Implement timeout
	args := []string{"install", "mysql-operator",
		"--debug",
		"--name", namespace,
		"--set", "operator.namespace=" + namespace,
		"--set", "image.tag=" + TestContext.OperatorVersion,
		"--set", "rbac.enabled=true", // TODO(apryde): Flag?
		"--set", "operator.global=false",
		"--set", "operator.register_crd=false"}
	Logf("Execing: %q", "helm "+strings.Join(args, " "))
	cmd := exec.Command("helm", args...)

	var err error
	cmd.Dir, err = filepath.Abs(TestContext.RepoRoot)
	if err != nil {
		return errors.Wrap(err, "getting abs path to repo root")
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		Logf("helm output: \n%s", string(output))
		return errors.Wrap(err, "installing operator")
	}

	By("Waiting for the operator to start")
	deploymentName := "mysql-operator"
	if err := wait.PollImmediate(Poll, DefaultTimeout, func() (bool, error) {
		deployment, err := f.ClientSet.AppsV1().Deployments(namespace).Get(deploymentName, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				return false, nil
			}
			return false, errors.Wrapf(err, "getting Deployment \"%s/%s\"", namespace, deploymentName)
		}

		for _, c := range deployment.Status.Conditions {
			if c.Type == extensionsv1betav1.DeploymentAvailable {
				return (c.Status == v1.ConditionTrue), nil
			}
		}
		return false, nil
	}); err != nil {
		return errors.New("timed out waiting for operator Pod")
	}

	return nil
}

// BeforeEach gets a client and makes a namespace.
func (f *Framework) BeforeEach() {
	// The fact that we need this feels like a bug in ginkgo.
	// https://github.com/onsi/ginkgo/issues/222
	f.cleanupHandle = AddCleanupAction(f.AfterEach)

	if f.ClientSet == nil {
		By("Creating a kubernetes client")
		config, err := clientcmd.BuildConfigFromFlags("", TestContext.KubeConfig)
		Expect(err).NotTo(HaveOccurred())
		f.ClientSet, err = clientset.NewForConfig(config)
		Expect(err).NotTo(HaveOccurred())
	}

	if f.MySQLClientSet == nil {
		By("Creating a MySQL Operator client")
		config, err := clientcmd.BuildConfigFromFlags("", TestContext.KubeConfig)
		Expect(err).NotTo(HaveOccurred())
		f.MySQLClientSet, err = mysqlclientset.NewForConfig(config)
		Expect(err).NotTo(HaveOccurred())
	}

	if TestContext.Namespace == "" {
		By("Building a namespace api object")
		namespace, err := f.CreateNamespace(f.BaseName, map[string]string{
			"e2e-framework": f.BaseName,
		})
		Expect(err).NotTo(HaveOccurred())
		f.Namespace = namespace
	} else {
		By(fmt.Sprintf("Getting existing namespace %q", TestContext.Namespace))
		namespace, err := f.ClientSet.CoreV1().Namespaces().Get(TestContext.Namespace, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		f.Namespace = namespace
	}

	if !f.OperatorInstalled {
		err := f.InstallOperator(f.Namespace.Name)
		Expect(err).NotTo(HaveOccurred())
		f.OperatorInstalled = true
	}
}

// AfterEach deletes the namespace(s).
func (f *Framework) AfterEach() {
	RemoveCleanupAction(f.cleanupHandle)

	if err := f.outputLogs(); err != nil {
		Logf("Failed to output container logs: %v", err)
	}

	nsDeletionErrors := map[string]error{}

	// Whether to delete namespace is determined by 3 factors: delete-namespace flag, delete-namespace-on-failure flag and the test result
	// if delete-namespace set to false, namespace will always be preserved.
	// if delete-namespace is true and delete-namespace-on-failure is false, namespace will be preserved if test failed.
	if TestContext.DeleteNamespace && (TestContext.DeleteNamespaceOnFailure || !CurrentGinkgoTestDescription().Failed) {
		for _, ns := range f.namespacesToDelete {
			By(fmt.Sprintf("Destroying namespace %q for this suite.", ns.Name))
			if err := f.DeleteNamespace(ns.Name, 5*time.Minute); err != nil {
				nsDeletionErrors[ns.Name] = err
			}
		}
	}

	// if we had errors deleting, report them now.
	if len(nsDeletionErrors) != 0 {
		messages := []string{}
		for namespaceKey, namespaceErr := range nsDeletionErrors {
			messages = append(messages, fmt.Sprintf("Couldn't delete ns: %q: %s (%#v)", namespaceKey, namespaceErr, namespaceErr))
		}
		Failf(strings.Join(messages, ","))
	}
	f.OperatorInstalled = false
}

func (f *Framework) outputLogs() error {
	pods, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).List(metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "listing test Pods")
	}

	var opPod v1.Pod
	var agPods []v1.Pod
	for _, pod := range pods.Items {
		if strings.Contains(pod.Spec.Containers[0].Image, "mysql-operator") {
			opPod = pod
			continue
		}
		if strings.Contains(pod.Spec.Containers[0].Image, "mysql-agent") || strings.Contains(pod.Spec.Containers[0].Image, "mysql-server") {
			agPods = append(agPods, pod)
		}
	}

	// Operator Logs
	if opPod.Name != "" {
		if err := f.printContainerLogs(opPod.GetName(), &v1.PodLogOptions{}); err != nil {
			return errors.Wrapf(err, "exporting mysql operator container logs for %s", opPod.GetName())
		}
	} else {
		Logf("MySQL Operator Pod could not be found. Logs have not been exported.")
	}

	for _, agPod := range agPods {
		// Server Logs
		if err := f.printContainerLogs(agPod.GetName(), &v1.PodLogOptions{Container: "mysql"}); err != nil {
			return errors.Wrapf(err, "exporting mysql server container logs for %s", agPod.GetName())
		}
		// Agent Logs
		if err := f.printContainerLogs(agPod.GetName(), &v1.PodLogOptions{Container: "mysql-agent"}); err != nil {
			return errors.Wrapf(err, "exporting mysql agent container logs for %s", agPod.GetName())
		}
	}

	return nil
}

func (f *Framework) printLogs(read io.ReadCloser, filepath string) error {
	defer read.Close()
	dst := os.Stdout
	if f.werckerReportArtifactsDir != "" {
		file, err := os.OpenFile(filepath, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			return errors.Wrapf(err, "opening log file %q", filepath)
		}
		dst = file
		defer dst.Close()
	}
	_, err := io.Copy(dst, read)
	if err != nil {
		var s string
		if filepath != "" {
			s = filepath
		} else {
			s = "stdout"
		}
		return errors.Wrapf(err, "writing logs to %q", s)
	}
	return nil
}

func (f *Framework) printContainerLogs(podName string, options *v1.PodLogOptions) error {
	podLogs := f.ClientSet.CoreV1().Pods(f.Namespace.Name).GetLogs(podName, options)
	if podLogs != nil {
		Logf("Writing %s container logs to file for %s", options.Container, podName)
		read, err := podLogs.Stream()
		if err != nil {
			return errors.Wrapf(err, "streaming request response for %s", podName)
		}
		f.printLogs(read, fmt.Sprintf("%s/%s%s", f.werckerReportArtifactsDir, podName, ".log"))
		Logf("Finished writing %s container logs for %s", options.Container, podName)
	}
	return nil
}
