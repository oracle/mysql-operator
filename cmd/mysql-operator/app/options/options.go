package options

import (
	"os"
	"time"

	"github.com/golang/glog"
	"github.com/spf13/pflag"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MySQLOperatorServer holds the options for the MySQLOperator.
type MySQLOperatorServer struct {
	// KubeConfig is the path to a kubeconfig file, specifying how to connect to
	// the API server.
	KubeConfig string
	// Master is the address of the Kubernetes API server (overrides any value
	// in kubeconfig).
	Master string

	// Namespace is the (optional) namespace in which the MySQL operator will
	// manage MySQL Clusters. Defaults to metav1.NamespaceAll.
	Namespace string

	// Hostname of the pod the operator is running in.
	Hostname string

	// minResyncPeriod is the resync period in reflectors; will be random
	// between minResyncPeriod and 2*minResyncPeriod.
	MinResyncPeriod metav1.Duration
}

// NewMySQLOperatorServer creates a new MySQLOperatorServer with defaults.
func NewMySQLOperatorServer() *MySQLOperatorServer {
	hostname, err := os.Hostname()
	if err != nil {
		glog.Fatalf("Failed to get the hostname: %v", err)
	}
	return &MySQLOperatorServer{
		MinResyncPeriod: metav1.Duration{Duration: 12 * time.Hour},
		Namespace:       metav1.NamespaceAll,
		Hostname:        hostname,
	}
}

// AddFlags adds the mysql-operator flags to a given FlagSet.
func (s *MySQLOperatorServer) AddFlags(fs *pflag.FlagSet) *pflag.FlagSet {
	fs.StringVar(&s.KubeConfig, "kubeconfig", s.KubeConfig, "Path to Kubeconfig file with authorization and master location information.")
	fs.StringVar(&s.Master, "master", s.Master, "The address of the Kubernetes API server (overrides any value in kubeconfig).")

	fs.StringVar(&s.Namespace, "namespace", s.Master, "The namespace for which the MySQL operator manages MySQL clusters. Defaults to all.")

	fs.DurationVar(&s.MinResyncPeriod.Duration, "min-resync-period", s.MinResyncPeriod.Duration, "The resync period in reflectors will be random between MinResyncPeriod and 2*MinResyncPeriod.")

	return fs
}
