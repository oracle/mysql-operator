package mysqldump

import (
	"fmt"

	"github.com/oracle/mysql-operator/pkg/apis/mysql/v1"
)

// Config holds the MySQL credentials required to authenticate with the MySQL database being
// backed-up or restored.
type Config struct {
	username  string
	password  string
	databases []string
}

// NewConfig creates an mysqldump configuration based on the input parameters.
func NewConfig(executor *v1.Executor, creds map[string]string) *Config {
	return &Config{
		databases: executor.Databases,
		username:  creds["username"],
		password:  creds["password"],
	}
}

// Validate checks the required configuration parameters are set.
func (c Config) Validate() (err error) {
	if c.username == "" {
		return fmt.Errorf("no mysqldump 'username' provided")
	}
	if c.password == "" {
		return fmt.Errorf("no mysqldump 'password' provided")
	}
	if c.databases == nil || len(c.databases) == 0 {
		return fmt.Errorf("no mysqldump 'databases' provided")
	}
	return nil
}
