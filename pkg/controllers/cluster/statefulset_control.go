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

	apps "k8s.io/api/apps/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubernetes "k8s.io/client-go/kubernetes"
	appslisters "k8s.io/client-go/listers/apps/v1beta1"

	"github.com/oracle/mysql-operator/pkg/constants"
	"github.com/oracle/mysql-operator/pkg/controllers/util"
	"github.com/oracle/mysql-operator/pkg/resources/statefulsets"
)

// StatefulSetControlInterface defines the interface that the
// MySQLClusterController uses to create, update, and delete StatefulSets. It
// is implemented as an interface to enable testing.
type StatefulSetControlInterface interface {
	CreateStatefulSet(ss *apps.StatefulSet) error
	DeleteStatefulSet(ss *apps.StatefulSet) error
	PatchStatefulSet(old *apps.StatefulSet, new *apps.StatefulSet) error
}

type realStatefulSetControl struct {
	client            kubernetes.Interface
	statefulSetLister appslisters.StatefulSetLister
}

// NewRealStatefulSetControl creates a concrete implementation of the
// StatefulSetControlInterface.
func NewRealStatefulSetControl(client kubernetes.Interface, statefulSetLister appslisters.StatefulSetLister) StatefulSetControlInterface {
	return &realStatefulSetControl{client: client, statefulSetLister: statefulSetLister}
}

func (rssc *realStatefulSetControl) CreateStatefulSet(ss *apps.StatefulSet) error {
	_, err := rssc.client.AppsV1beta1().StatefulSets(ss.Namespace).Create(ss)
	return err
}

func (rssc *realStatefulSetControl) DeleteStatefulSet(ss *apps.StatefulSet) error {
	policy := metav1.DeletePropagationBackground
	opts := &metav1.DeleteOptions{PropagationPolicy: &policy}
	err := rssc.client.AppsV1beta1().StatefulSets(ss.Namespace).Delete(ss.Name, opts)
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

func (rssc *realStatefulSetControl) PatchStatefulSet(old *apps.StatefulSet, new *apps.StatefulSet) error {
	_, err := util.PatchStatefulSet(rssc.client, old, new)
	return err
}

// updateStatefulSetToOperatorVersion sets the specified MySQLOperator version on:
//   1. The StatefulSet operator version label.
//   2. The MySQLAgent container image version
func updateStatefulSetToOperatorVersion(ss *apps.StatefulSet, mysqlAgentImage string, version string) *apps.StatefulSet {
	targetContainer := statefulsets.MySQLAgentName
	newAgentImage := fmt.Sprintf("%s:%s", mysqlAgentImage, version)
	ss.ObjectMeta.Labels[constants.MySQLOperatorVersionLabel] = version
	for idx, container := range ss.Spec.Template.Spec.Containers {
		if container.Name == targetContainer {
			ss.Spec.Template.Spec.Containers[idx].Image = newAgentImage
			break
		}
	}
	return ss
}
