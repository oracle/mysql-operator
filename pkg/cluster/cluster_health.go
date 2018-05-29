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
	"sync"

	"github.com/heptiolabs/healthcheck"
	"github.com/pkg/errors"

	"github.com/oracle/mysql-operator/pkg/cluster/innodb"
)

var (
	status      *innodb.ClusterStatus
	statusMutex sync.Mutex
)

// SetStatus sets the status of the local mysql cluster. The cluster manager
// controller is responsible for updating.
func SetStatus(new *innodb.ClusterStatus) {
	statusMutex.Lock()
	defer statusMutex.Unlock()
	status = new.DeepCopy()
}

// GetStatus fetches a copy of the latest cluster status.
func GetStatus() *innodb.ClusterStatus {
	statusMutex.Lock()
	defer statusMutex.Unlock()
	if status == nil {
		return nil
	}
	return status.DeepCopy()
}

// NewHealthCheck constructs a healthcheck for the local instance which checks
// cluster status using mysqlsh.
func NewHealthCheck() (healthcheck.Check, error) {
	instance, err := NewLocalInstance()
	if err != nil {
		return nil, errors.Wrap(err, "getting local mysql instance")
	}

	return func() error {
		s := GetStatus()
		if s == nil || s.GetInstanceStatus(instance.Name()) != innodb.InstanceStatusOnline {
			return errors.New("database still requires management")
		}
		return nil
	}, nil
}
