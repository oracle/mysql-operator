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

package innodb

import (
	"fmt"
	"net"
)

// DefaultClusterName is the default name assigned to InnoDB clusters created by
// the MySQL operator.
const DefaultClusterName = "Cluster"

// MySQLDBPort is port on which MySQL listens for client connections.
const MySQLDBPort = 3306

// InstanceStatus denotes the status of a MySQL Instance.
type InstanceStatus string

// Instance statuses.
const (
	InstanceStatusOnline      InstanceStatus = "ONLINE"
	InstanceStatusMissing                    = "(MISSING)"
	InstanceStatusRecovering                 = "RECOVERING"
	InstanceStatusUnreachable                = "UNREACHABLE"
	InstanceStatusNotFound                   = ""
	InstanceStatusUnknown                    = "UNKNOWN"
)

// instanceState denotes the state of a MySQL Instance.
type instanceState string

// Instance states.
const (
	instanceStateOk    instanceState = "ok"
	instanceStateError               = "error"
)

// instanceReason denotes the reason for the state of a MySQL Instance.
type instanceReason string

// Instance reasons.
const (
	instanceReasonRecoverable instanceReason = "recoverable"
)

// InstanceMode denotes the mode of a MySQL Instance.
type InstanceMode string

// Instance modes.
const (
	ReadWrite InstanceMode = "R/W"
	ReadOnly               = "R/O"
)

// Instance represents an individual MySQL instance in an InnoDB cluster.
type Instance struct {
	Address string         `json:"address"`
	Mode    InstanceMode   `json:"mode"`
	Role    string         `json:"role"`
	Status  InstanceStatus `json:"status"`
}

// InstanceState represents the state of a MySQL instance with respect to an
// InnoDB cluster.
type InstanceState struct {
	Reason instanceReason `json:"reason"`
	State  instanceState  `json:"state"`
}

// ReplicaSetStatus denotes the state of a MySQL replica set.
type ReplicaSetStatus string

// Replica set statuses
const (
	ReplicaSetStatusOk            ReplicaSetStatus = "OK"
	ReplicaSetStatusOkPartial                      = "OK_PARTIAL"
	ReplicaSetStatusOkNoTolerance                  = "OK_NO_TOLERANCE"
	ReplicaSetStatusNoQuorum                       = "NO_QUORUM"
	ReplicaSetStatusUnknown                        = "UNKNOWN"
)

// ReplicaSet holds the server instances which belong to an InnoDB
// cluster.
type ReplicaSet struct {
	Name       string               `json:"name"`
	Primary    string               `json:"primary"`
	Status     ReplicaSetStatus     `json:"status"`
	StatusText string               `json:"statusText"`
	Topology   map[string]*Instance `json:"topology"`
}

// DeepCopy takes a deep copy of a ReplicaSet object.
func (rs *ReplicaSet) DeepCopy() *ReplicaSet {
	new := new(ReplicaSet)
	*new = *rs
	for k := range rs.Topology {
		new.Topology[k] = rs.Topology[k].DeepCopy()
	}
	return new
}

// ClusterStatus represents the status of an InnoDB cluster
type ClusterStatus struct {
	ClusterName       string     `json:"clusterName"`
	DefaultReplicaSet ReplicaSet `json:"defaultReplicaSet"`
}

// GetInstanceStatus returns the InstanceStatus of the given instance.
func (s *ClusterStatus) GetInstanceStatus(name string) InstanceStatus {
	if s.DefaultReplicaSet.Topology == nil {
		return InstanceStatusNotFound
	}
	if is, ok := s.DefaultReplicaSet.Topology[fmt.Sprintf("%s:%d", name, MySQLDBPort)]; ok {
		return is.Status
	}
	return InstanceStatusNotFound
}

// GetPrimaryAddr returns a primary in the given cluster.
func (s *ClusterStatus) GetPrimaryAddr() (string, error) {
	if s.DefaultReplicaSet.Primary != "" {
		// Single-primary mode.
		return s.DefaultReplicaSet.Primary, nil
	}
	for _, instance := range s.DefaultReplicaSet.Topology {
		// Multi-primary mode.
		if instance.Mode == ReadWrite {
			return instance.Address, nil
		}
	}
	return "", fmt.Errorf("unable to find primary for cluster: %s", s.ClusterName)
}

// DeepCopy takes a deep copy of a ClusterStatus object.
func (s *ClusterStatus) DeepCopy() *ClusterStatus {
	new := new(ClusterStatus)
	*new = *s
	new.DefaultReplicaSet = *s.DefaultReplicaSet.DeepCopy()
	return new
}

// Name returns the dns name of the Instance.
func (i *Instance) Name() string {
	name, _, _ := net.SplitHostPort(i.Address)
	return name
}

// DeepCopy takes a deep copy of an Instance object.
func (i *Instance) DeepCopy() *Instance {
	new := new(Instance)
	*new = *i
	new.Status = i.Status
	return new
}

// CanRejoinCluster returns true if the instance can rejoin the InnoDB cluster.
func (s *InstanceState) CanRejoinCluster() bool {
	return s.State == instanceStateOk && s.Reason == instanceReasonRecoverable
}
