package s3

// baremetal "github.com/oracle/bmcs-go-sdk"

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	s3 "github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/pkg/errors"
)

// Client is an S3 client and configured Uploader.
type Client struct {
	s3         *s3.S3
	s3Uploader *s3manager.Uploader
}

// NewClient constructs a new S3 backup upload provider client that can upload/download
// backups to any S3 compliant API e.g. OCI, AWS, GCE.
func NewClient(config *Config) (*Client, error) {
	s3Config := aws.NewConfig().
		WithCredentials(credentials.NewStaticCredentials(config.accessKey, config.secretKey, "")).
		WithEndpoint(config.endpoint).
		WithRegion(config.region).
		WithS3ForcePathStyle(true)

	sess, err := getSession(s3Config)
	if err != nil {
		return nil, err
	}
	s3 := s3.New(sess)
	s3Uploader := s3manager.NewUploader(sess)

	return &Client{s3: s3, s3Uploader: s3Uploader}, nil
}

func getSession(config *aws.Config) (*session.Session, error) {
	sess, err := session.NewSession(config)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if _, err := sess.Config.Credentials.Get(); err != nil {
		return nil, errors.WithStack(err)
	}

	return sess, nil
}
