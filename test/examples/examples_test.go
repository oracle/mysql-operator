package examples

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	yaml "github.com/ghodss/yaml"

	api "github.com/oracle/mysql-operator/pkg/apis/mysql/v1"

	corev1 "k8s.io/api/core/v1"
)

func TestRemoveInstanceFromCluster(t *testing.T) {
	fileList := findYamlFiles(t, "../../examples/")
	for _, file := range fileList {
		kind := getKind(t, file)
		t.Logf("validating file: %s of kind: %v", file, kind)
		switch kind {
		case "MySQLCluster":
			validateMySQLCluster(t, file)
		case "MySQLBackup":
			validateMySQLBackup(t, file)
		case "MySQLRestore":
			validateMySQLRestore(t, file)
		case "MySQLBackupSchedule":
			validateMySQLBackupSchedule(t, file)
		default:
			t.Logf("ignoring file: %s of kind: %v", file, kind)
		}
	}
}

func validateMySQLCluster(t *testing.T, file string) {
	bytes, err := ioutil.ReadFile(file)
	if err != nil {
		t.Errorf("Failed to read file: %s, err: %v", file, err)
		return
	}
	var r api.MySQLCluster
	err = yaml.Unmarshal(bytes, &r)
	if err != nil {
		t.Errorf("Failed to parse file: %s, err: %v", file, err)
		return
	}
	resource := r.EnsureDefaults()
	err = resource.Validate()
	if err != nil {
		t.Errorf("Failed to validate file: %s, err: %v", file, err)
		return
	}
}

func validateMySQLBackup(t *testing.T, file string) {
	bytes, err := ioutil.ReadFile(file)
	if err != nil {
		t.Errorf("Failed to read file: %s, err: %v", file, err)
		return
	}
	var r api.MySQLBackup
	err = yaml.Unmarshal(bytes, &r)
	if err != nil {
		t.Errorf("Failed to parse file: %s, err: %v", file, err)
		return
	}
	r.Spec.ClusterRef = &corev1.LocalObjectReference{}
	r.Spec.Storage.SecretRef = &corev1.LocalObjectReference{Name: "test"}
	resource := r.EnsureDefaults()
	err = resource.Validate()
	if err != nil {
		t.Errorf("Failed to validate: %s, err: %v", file, err)
		return
	}
}

func validateMySQLRestore(t *testing.T, file string) {
	bytes, err := ioutil.ReadFile(file)
	if err != nil {
		t.Errorf("Failed to read file: %s, err: %v", file, err)
		return
	}
	var r api.MySQLRestore
	err = yaml.Unmarshal(bytes, &r)
	if err != nil {
		t.Errorf("Failed to parse file: %s, err: %v", file, err)
		return
	}
	resource := r.EnsureDefaults()
	err = resource.Validate()
	if err != nil {
		t.Errorf("Failed to validate: %s, err: %v", file, err)
		return
	}
}

func validateMySQLBackupSchedule(t *testing.T, file string) {
	bytes, err := ioutil.ReadFile(file)
	if err != nil {
		t.Errorf("Failed to read file: %s, err: %v", file, err)
		return
	}
	var r api.MySQLBackupSchedule
	err = yaml.Unmarshal(bytes, &r)
	if err != nil {
		t.Errorf("Failed to parse file: %s, err: %v", file, err)
		return
	}
	resource := r.EnsureDefaults()
	err = resource.Validate()
	if err != nil {
		t.Errorf("Failed to validate: %s, err: %v", file, err)
		return
	}
}

func findYamlFiles(t *testing.T, searchDir string) []string {
	fileList := []string{}
	err := filepath.Walk(searchDir, func(path string, f os.FileInfo, err error) error {
		if strings.HasSuffix(path, ".yaml") {
			fileList = append(fileList, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to find the list of all yaml files, err: %v", err)
	}
	return fileList
}

type Resource struct {
	Kind string `json:"kind"`
}

func getKind(t *testing.T, file string) string {
	bytes, err := ioutil.ReadFile(file)
	if err != nil {
		t.Fatalf("Failed to read yaml file, err: %v", err)
	}
	var r Resource
	err = yaml.Unmarshal(bytes, &r)
	if err != nil {
		t.Fatalf("Failed to parse yaml file, err: %v", err)
	}
	return r.Kind
}
