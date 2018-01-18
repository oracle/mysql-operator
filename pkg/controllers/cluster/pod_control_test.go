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
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/oracle/mysql-operator/pkg/controllers/util"
)

type fakePodControl struct {
	PodControlInterface
	client kubernetes.Interface
}

// NewFakePodControl creates a concrete FAKE implementation of the PodControlInterface.
func NewFakePodControl(podControl PodControlInterface, client kubernetes.Interface) PodControlInterface {
	return &fakePodControl{podControl, client}
}

func (rpc *fakePodControl) PatchPod(old *v1.Pod, new *v1.Pod) error {
	_, err := util.UpdatePod(rpc.client, new)
	return err
}
