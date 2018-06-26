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

package manager

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/oracle/mysql-operator/pkg/cluster"
)

func TestGetReplicationGroupSeeds(t *testing.T) {
	testCases := []struct {
		seeds    string
		pod      *cluster.Instance
		expected []string
	}{
		{
			seeds:    "server-1-0:1234,server-1-1:1234",
			pod:      cluster.NewInstance("", "", "server-1", 0, -1, false),
			expected: []string{"server-1-1:1234"},
		}, {
			seeds:    "server-1-1:1234,server-1-0:1234",
			pod:      cluster.NewInstance("", "", "server-1", 0, -1, false),
			expected: []string{"server-1-1:1234"},
		}, {
			seeds:    "server-1-0:1234,server-1-1:1234",
			pod:      cluster.NewInstance("", "", "server-2", 0, -1, false),
			expected: []string{"server-1-0:1234", "server-1-1:1234"},
		}, {
			seeds:    "server-1-0.server-1:1234,server-1-1.server-1:1234",
			pod:      cluster.NewInstance("", "", "server-2", 0, -1, false),
			expected: []string{"server-1-0.server-1:1234", "server-1-1.server-1:1234"},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.seeds, func(t *testing.T) {
			output, err := getReplicationGroupSeeds(tt.seeds, tt.pod)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, output)
		})
	}
}
