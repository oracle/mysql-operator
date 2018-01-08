package e2e

import (
	"os"
	"testing"

	"github.com/golang/glog"

	"github.com/oracle/mysql-operator/test/e2e/framework"
)

func TestMain(m *testing.M) {
	if err := framework.Setup(); err != nil {
		glog.Errorf("Failed to setup framework: %v", err)
		os.Exit(1)
	}

	code := m.Run()

	if err := framework.Teardown(); err != nil {
		glog.Errorf("Failed to teardown framework: %v", err)
		os.Exit(1)
	}
	os.Exit(code)
}
