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
	corelisters "k8s.io/client-go/listers/core/v1"
)

// ServiceControlInterface defines the interface that the MySQLClusterController
// uses to create Services. It is implemented as an interface to enable testing.
type ServiceControlInterface interface {
	CreateService(s *v1.Service) error
}

type realServiceControl struct {
	client        kubernetes.Interface
	serviceLister corelisters.ServiceLister
}

// NewRealServiceControl creates a concrete implementation of the
// ServiceControlInterface.
func NewRealServiceControl(client kubernetes.Interface, serviceLister corelisters.ServiceLister) ServiceControlInterface {
	return &realServiceControl{client: client, serviceLister: serviceLister}
}

func (rsc *realServiceControl) CreateService(s *v1.Service) error {
	_, err := rsc.client.CoreV1().Services(s.Namespace).Create(s)
	return err
}
