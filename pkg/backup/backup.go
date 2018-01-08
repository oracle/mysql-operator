package backup

import (
	"os"

	"github.com/oracle/mysql-operator/pkg/apis/mysql/v1"
	"github.com/oracle/mysql-operator/pkg/backup/executor"
	"github.com/oracle/mysql-operator/pkg/backup/storage"
	"github.com/oracle/mysql-operator/pkg/resources/statefulsets"
)

const (
	backupDir  = statefulsets.MySQLAgentBasePath + "/backup"
	restoreDir = statefulsets.MySQLAgentBasePath + "/restore"
)

// Runner implementations can execute backups and store them in storage
// backends.
type Runner interface {
	Backup(clusterName string) (string, error)
	Restore(key string) error
}

type runner struct {
	executor executor.Interface
	storage  storage.Interface
}

// NewConfiguredRunner creates a runner configured with the Backup/Restore target executor and
// storage configurations.
func NewConfiguredRunner(execConfig *v1.Executor, execCreds map[string]string, storeConfig *v1.Storage, storeCreds map[string]string) (Runner, error) {
	exec, err := executor.New(execConfig, execCreds)
	if err != nil {
		return nil, err
	}

	store, err := storage.NewStorageProvider(storeConfig, storeCreds)
	if err != nil {
		return nil, err
	}

	return &runner{executor: exec, storage: store}, nil
}

// Backup performs a backup using the executor and then stores it using the storage provider.
func (r *runner) Backup(clusterName string) (string, error) {
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		if err := os.MkdirAll(backupDir, os.ModePerm); err != nil {
			return "", err
		}
	}
	defer os.RemoveAll(backupDir)

	reader, key, err := r.executor.Backup(backupDir, clusterName)
	if err != nil {
		return "", err
	}

	err = r.storage.Store(key, reader)
	if err != nil {
		return "", err
	}
	return key, nil
}

// Restore performs a retrieve using the storage providor then a restore using
// the executor.
func (r *runner) Restore(key string) error {
	if _, err := os.Stat(restoreDir); os.IsNotExist(err) {
		if err := os.MkdirAll(restoreDir, os.ModePerm); err != nil {
			return err
		}
	}
	defer os.RemoveAll(restoreDir)

	reader, err := r.storage.Retrieve(key)
	if err != nil {
		return err
	}

	err = r.executor.Restore(reader)
	if err != nil {
		return err
	}

	return nil
}
