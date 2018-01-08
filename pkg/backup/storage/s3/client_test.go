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
