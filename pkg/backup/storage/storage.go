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

package storage

import (
	"fmt"
	"io"
	"strings"

	"github.com/oracle/mysql-operator/pkg/apis/mysql/v1alpha1"
	"github.com/oracle/mysql-operator/pkg/backup/storage/s3"
)

const (
	// ProviderS3 denotes S3 compatability backed storage provider.
	ProviderS3 = "s3"
)

// Interface abstracts the underlying storage provider.
type Interface interface {
	// Store creates a new object in the underlying provider's datastore if it does not exist,
	// or replaces the existing object if it does exist.
	Store(key string, body io.ReadCloser) error
	// Retrieve return the object in the underlying provider's datastore if it exists.
	Retrieve(key string) (io.ReadCloser, error)
}

// NewStorageProvider accepts a secret map and uses its contents to determine the
// desired object storage provider implementation.
func NewStorageProvider(config *v1alpha1.Storage, creds map[string]string) (Interface, error) {
	switch strings.ToLower(config.Provider) {
	case ProviderS3:
		return s3.NewStorage(config, creds)
	default:
		return nil, fmt.Errorf("unknown backup storage provider %q", config.Provider)
	}
}
