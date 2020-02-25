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
	"github.com/oracle/mysql-operator/pkg/controllers/util"
	appsv1beta1 "k8s.io/api/apps/v1"
	kubernetes "k8s.io/client-go/kubernetes"
)

type fakeStatefulSetControl struct {
	StatefulSetControlInterface
	client kubernetes.Interface
}

// NewFakeStatefulSetControl creates a concrete FAKE implementation of the StatefulSetControlInterface.
func NewFakeStatefulSetControl(statefulSetControl StatefulSetControlInterface, client kubernetes.Interface) StatefulSetControlInterface {
	return &fakeStatefulSetControl{statefulSetControl, client}
}

func (rssc *fakeStatefulSetControl) Patch(old *appsv1beta1.StatefulSet, new *appsv1beta1.StatefulSet) error {
	_, err := util.UpdateStatefulSet(rssc.client, new)
	return err
}
