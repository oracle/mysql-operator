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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeepCopyInstance(t *testing.T) {
	testCases := []struct {
		name string
		in   *Instance
	}{
		{
			name: "all-fields-populated",
			in: &Instance{
				Address: "localhost:3310",
				Mode:    "R/O",
				Role:    "HA",
				Status:  InstanceStatusOnline,
			},
		},
		{
			name: "all-fields-empty",
			in:   &Instance{},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			copy := tt.in.DeepCopy()
			assert.Equal(t, tt.in, copy)

			// Modify copy and check it is no longer equal to input (i.e. input
			// has not been mutated).
			copy.Address = "9.9.9.9"
			assert.NotEqual(t, tt.in, copy)
		})
	}
}

func TestDeepCopyReplicaSet(t *testing.T) {
	testCases := []struct {
		name string
		in   *ReplicaSet
	}{
		{
			name: "all-fields-populated",
			in: &ReplicaSet{
				Name:    "default",
				Primary: "localhost:3310",
				Status:  "OK",
				Topology: map[string]*Instance{
					"localhost:3310": {
						Address: "localhost:3310",
						Mode:    "R/O",
						Role:    "HA",
						Status:  InstanceStatusOnline,
					},
				},
			},
		},
		{
			name: "all-fields-empty",
			in:   &ReplicaSet{},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			copy := tt.in.DeepCopy()
			assert.Equal(t, tt.in, copy)

			// Modify copy and check it is no longer equal to input (i.e. input
			// has not been mutated).
			copy.Name = "non-default"
			assert.NotEqual(t, tt.in, copy)
		})
	}
}

func TestDeepCopyClusterStatus(t *testing.T) {
	testCases := []struct {
		name string
		in   *ClusterStatus
	}{
		{
			name: "all-fields-populated",
			in: &ClusterStatus{
				ClusterName: "MyCluster",
				DefaultReplicaSet: ReplicaSet{
					Name:    "MyCluster",
					Primary: "localhost:3310",
					Status:  "OK",
					Topology: map[string]*Instance{
						"localhost:3310": {
							Address: "localhost:3310",
							Mode:    "R/O",
							Role:    "HA",
							Status:  InstanceStatusOnline,
						},
					},
				},
			},
		},
		{
			name: "all-fields-empty",
			in:   &ClusterStatus{},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			copy := tt.in.DeepCopy()
			assert.Equal(t, tt.in, copy)

			// Modify copy and check it is no longer equal to input (i.e. input
			// has not been mutated).
			copy.ClusterName = "YourCluster"
			assert.NotEqual(t, tt.in, copy)
		})
	}
}
