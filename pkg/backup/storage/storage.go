package storage

import (
	"fmt"
	"io"
	"strings"

	"github.com/oracle/mysql-operator/pkg/apis/mysql/v1"
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
func NewStorageProvider(config *v1.Storage, creds map[string]string) (Interface, error) {
	switch strings.ToLower(config.Provider) {
	case ProviderS3:
		return s3.NewStorage(config, creds)
	default:
		return nil, fmt.Errorf("unknown backup storage provider %q", config.Provider)
	}
}
