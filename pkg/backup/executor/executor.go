package executor

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/oracle/mysql-operator/pkg/apis/mysql/v1"
	"github.com/oracle/mysql-operator/pkg/backup/executor/mysqldump"
)

const (
	// MySQLDumpProvider denotes the mysqldump utility backup and restore provider.
	MySQLDumpProvider = "mysqldump"
)

// ExecutorProviders denotes the list of available ExecutorProviders.
var ExecutorProviders = [...]string{MySQLDumpProvider}

// Interface will execute backup operations via a tool such as mysqlbackup or
// mysqldump.
type Interface interface {
	// Backup runs a backup operation using the given credentials, returning the content.
	// TODO: default backupDir to allow streaming...
	Backup(backupDir string, clusterName string) (io.ReadCloser, string, error)
	// Restore restores the given content to the mysql node.
	Restore(content io.ReadCloser) error
}

// New builds a new backup executor.
func New(executor *v1.Executor, creds map[string]string) (Interface, error) {
	switch strings.ToLower(executor.Provider) {
	case MySQLDumpProvider:
		return mysqldump.NewExecutor(executor, creds)
	default:
		return nil, fmt.Errorf("unknown backup executor provider %q", executor.Provider)
	}
}

// DefaultCreds return the default MySQL credentials for the local instance.
func DefaultCreds() map[string]string {
	return map[string]string{
		"username": "root",
		"password": os.Getenv("MYSQL_ROOT_PASSWORD"),
	}
}
