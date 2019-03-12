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

package oss

import (
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	"io"

	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/oracle/mysql-operator/pkg/apis/mysql/v1alpha1"
)

// Provider is storage implementation of provider.Interface.
type Provider struct {
	v1alpha1.S3StorageProvider

	oss *oss.Client
}

// NewProvider creates a new S3 (compatible) storage provider.
func NewProvider(provider *v1alpha1.S3StorageProvider, credentials map[string]string) (*Provider, error) {
	accessKey, secretKey, err := getCredentials(credentials)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	client, err := oss.New(provider.Endpoint, accessKey, secretKey)

	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &Provider{
		S3StorageProvider: *provider,
		oss:               client,
	}, nil
}

// Store the given data at the given key.
func (p *Provider) Store(key string, body io.ReadCloser) error {
	glog.V(2).Infof("Storing backup (provider=\"S3\", endpoint=%q, bucket=%q, key=%q)", p.Endpoint, p.Bucket, key)

	defer body.Close()

	bucket, err := p.oss.Bucket(p.Bucket)
	if err != nil {
		return errors.Wrapf(err, "error storing backup (provider=\"S3\", endpoint=%q, bucket=%q, key=%q)", p.Endpoint, p.Bucket, key)
	}
	err = bucket.PutObject(key, body)
	return errors.Wrapf(err, "error storing backup (provider=\"S3\", endpoint=%q, bucket=%q, key=%q)", p.Endpoint, p.Bucket, key)
}

// Retrieve the given key from S3 storage service.
func (p *Provider) Retrieve(key string) (io.ReadCloser, error) {
	glog.V(2).Infof("Retrieving backup (provider=\"s3\", endpoint=%q, bucket=%q, key=%q)", p.Endpoint, p.Bucket, key)
	bucket, err := p.oss.Bucket(p.Bucket)
	if err != nil {
		return nil, errors.Wrapf(err, "error retrieving backup (provider='S3', endpoint='%s', bucket='%s', key='%s')", p.Endpoint, p.Bucket, key)
	}
	obj, err := bucket.GetObject(key)
	if err != nil {
		return nil, errors.Wrapf(err, "error retrieving backup (provider='S3', endpoint='%s', bucket='%s', key='%s')", p.Endpoint, p.Bucket, key)
	}

	return obj, nil
}

// getCredentials gets an accesskey and secretKey from the provided map.
func getCredentials(credentials map[string]string) (string, string, error) {
	allErrs := field.ErrorList{}
	fldPath := field.NewPath("data")

	if credentials == nil {
		return "", "", errors.New("no credentials provided")
	}

	accessKey, ok := credentials["accessKey"]
	if !ok {
		allErrs = append(allErrs, field.Required(fldPath.Child("accessKey"), ""))
	}
	secretKey, ok := credentials["secretKey"]
	if !ok {
		allErrs = append(allErrs, field.Required(fldPath.Child("secretKey"), ""))
	}

	if len(allErrs) > 0 {
		return "", "", allErrs.ToAggregate()
	}

	return accessKey, secretKey, nil
}
