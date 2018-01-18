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
	"testing"
)

func TestGetParentNameAndOrdinal(t *testing.T) {
	testCases := []struct {
		hostname string
		name     string
		ordinal  int
	}{
		{
			hostname: "host-99",
			name:     "host",
			ordinal:  99,
		}, {
			hostname: "host-with-dashes-99",
			name:     "host-with-dashes",
			ordinal:  99,
		}, {
			hostname: "host_with_no_dashes",
			name:     "",
			ordinal:  -1,
		}, {
			hostname: "host-string_instead_of_ordinal",
			name:     "",
			ordinal:  -1,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.hostname, func(t *testing.T) {
			name, ordinal := getParentNameAndOrdinal(tt.hostname)
			if name != tt.name || ordinal != tt.ordinal {
				t.Errorf("getParentNameAndOrdinal(%q) => (%q, %d) expected (%q, %d)",
					tt.hostname, name, ordinal, tt.name, tt.ordinal)
			}
		})
	}
}
