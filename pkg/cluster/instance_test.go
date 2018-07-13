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
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
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
			name, ordinal := GetParentNameAndOrdinal(tt.hostname)
			if name != tt.name || ordinal != tt.ordinal {
				t.Errorf("getParentNameAndOrdinal(%q) => (%q, %d) expected (%q, %d)",
					tt.hostname, name, ordinal, tt.name, tt.ordinal)
			}
		})
	}
}

func TestGetPodName(t *testing.T) {
	testCases := []struct {
		seed string
		name string
	}{
		{
			seed: "host-99.host:3306",
			name: "host-99",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.seed, func(t *testing.T) {
			name, err := podNameFromSeed(tt.seed)
			assert.NoError(t, err)
			assert.Equal(t, tt.name, name)
		})
	}
}

func TestWhitelistCIDR(t *testing.T) {
	testCases := []struct {
		ip       string
		expected string
	}{
		{ip: "192.168.0.1", expected: "192.168.0.0/16"},
		{ip: "192.167.0.1", expected: ""},
		{ip: "10.1.1.1", expected: "10.0.0.0/8"},
		{ip: "172.15.0.1", expected: ""},
		{ip: "172.16.0.1", expected: "172.16.0.0/12"},
		{ip: "172.17.0.1", expected: "172.16.0.0/12"},
		{ip: "100.64.0.1", expected: "100.64.0.0/10"},
		{ip: "100.63.0.1", expected: ""},
		{ip: "1.2.3.4", expected: ""},
	}

	for _, tt := range testCases {
		i := Instance{IP: net.ParseIP(tt.ip)}

		cidr, _ := i.WhitelistCIDR()
		if cidr != tt.expected {
			t.Errorf("ip: %v, cidr: %v, expected: %v", tt.ip, cidr, tt.expected)
		}
	}
}
