package s3

import (
	"testing"

	"github.com/oracle/mysql-operator/pkg/apis/mysql/v1"
)

func TestConfigFromSecretDataWithValidData(t *testing.T) {
	storage := &v1.Storage{
		Provider: "s3",
		Config: map[string]string{
			"endpoint": "endpoint",
			"region":   "region",
			"bucket":   "bucket",
		},
	}

	creds := map[string]string{
		"accessKey": "accessKey",
		"secretKey": "secretKey",
	}

	config := NewConfig(storage, creds)

	err := config.Validate()
	if err != nil {
		t.Errorf("Expected config to be valid but got error: %+v", err)
	}
}

func TestConfigFromSecretDataWithInValidData(t *testing.T) {
	storage := &v1.Storage{
		Provider: "s3",
		Config: map[string]string{
			"endpoint": "endpoint",
			"region":   "region",
			"bucket":   "bucket",
		},
	}

	creds := map[string]string{}

	config := NewConfig(storage, creds)

	err := config.Validate()
	if err == nil {
		t.Error("Expected config to be invalid but was valid")
	}
}
