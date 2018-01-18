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
	"io"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/golang/glog"
	"github.com/pkg/errors"

	"github.com/oracle/mysql-operator/pkg/apis/mysql/v1"
)

// Provider is storage implementation of provider.Interface.
type Provider struct {
	client *Client
	config *Config
}

// NewStorage creates a provider capable of storing and retreiving objects against the specified
// 's3' storage configuration and credentials.
func NewStorage(config *v1.Storage, creds map[string]string) (*Provider, error) {
	cfg := NewConfig(config, creds)
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	c, err := NewClient(cfg)
	if err != nil {
		return nil, err
	}
	return &Provider{client: c, config: cfg}, nil
}

// Store will upload the content of the data stream to S3.
func (p *Provider) Store(key string, body io.ReadCloser) error {
	glog.V(4).Infof("storing backup (provider='s3', endpoint='%s', bucket='%s', key='%s')", p.config.endpoint, p.config.bucket, key)
	defer body.Close()
	rq := &s3manager.UploadInput{
		Bucket: &p.config.bucket,
		Key:    &key,
		Body:   body,
	}
	_, err := p.client.s3Uploader.Upload(rq)
	return errors.Wrapf(err, "error storing backup (provider='s3', endpoint='%s', bucket='%s', key='%s')", p.config.endpoint, p.config.bucket, key)
}

// Retrieve will provide a data stream on the specified object from S3.
func (p *Provider) Retrieve(key string) (io.ReadCloser, error) {
	glog.V(4).Infof("retrieving backup (provider='s3', endpoint='%s', bucket='%s', key='%s')", p.config.endpoint, p.config.bucket, key)
	req := &s3.GetObjectInput{Bucket: &p.config.bucket, Key: &key}
	obj, err := p.client.s3.GetObject(req)
	if err != nil {
		return nil, errors.Wrapf(err, "error retrieving backup (provider='s3', endpoint='%s', bucket='%s', key='%s')", p.config.endpoint, p.config.bucket, key)
	}
	return obj.Body, nil
}
