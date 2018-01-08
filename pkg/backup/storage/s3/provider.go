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
