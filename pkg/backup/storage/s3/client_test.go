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

package s3

import (
	"testing"

	"github.com/oracle/mysql-operator/pkg/apis/mysql/v1"
)

func TestClientWithInvalidData(t *testing.T) {
	storage := &v1.Storage{
		Provider: "s3",
		Config: map[string]string{
			"region": "region",
			"bucket": "bucket",
		},
	}

	creds := map[string]string{}

	config := NewConfig(storage, creds)

	client, err := NewClient(config)
	if err == nil {
		t.Error("Expected NewClient to be return an error on invalid config")
	}
	if client != nil {
		t.Error("Expected NewClient to be return an nil client on invalid config")
	}

}
