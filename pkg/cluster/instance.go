package cluster

import (
	"fmt"
	"net"
	"os"
	"regexp"
	"strconv"

	"github.com/oracle/mysql-operator/pkg/cluster/innodb"
)

// Instance represents the local MySQL instance.
type Instance struct {
	// Namespace is the Kubernetes Namespace in which the instance is running.
	Namespace string
	// ClusterName is the name of the MySQLCluster to which the instance
	// belongs.
	ClusterName string
	// ParentName is the name of the StatefulSet to which the instance belongs.
	ParentName string
	// Ordinal is the StatefulSet ordinal of the instances Pod.
	Ordinal int
	// Port is the port on which MySQLDB is listening.
	Port int
}

// NewInstance creates a new Instance.
func NewInstance(namespace, clusterName, parentName string, ordinal, port int) *Instance {
	return &Instance{
		Namespace:   namespace,
		ClusterName: clusterName,
		ParentName:  parentName,
		Ordinal:     ordinal,
		Port:        port,
	}
}

// NewLocalInstance creates a new instance of this structure, with it's name and index
// populated from os.Hostname().
func NewLocalInstance() (*Instance, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	name, ordinal := getParentNameAndOrdinal(hostname)
	return &Instance{
		Namespace:   os.Getenv("POD_NAMESPACE"),
		ClusterName: os.Getenv("MYSQL_CLUSTER_NAME"),
		ParentName:  name,
		Ordinal:     ordinal,
		Port:        innodb.MySQLDBPort,
	}, nil
}

// NewInstanceFromGroupSeed creates an Instance from a fully qualified group
// seed.
func NewInstanceFromGroupSeed(seed string) (*Instance, error) {
	// We don't care about the returned port here as the Instance's port its
	// MySQLDB port not its group replication port.
	host, _, err := net.SplitHostPort(seed)
	if err != nil {
		return nil, err
	}
	parentName, ordinal := getParentNameAndOrdinal(host)
	return &Instance{
		ClusterName: os.Getenv("MYSQL_CLUSTER_NAME"),
		Namespace:   os.Getenv("POD_NAMESPACE"),
		ParentName:  parentName,
		Ordinal:     ordinal,
		Port:        innodb.MySQLDBPort,
	}, nil
}

// GetUser returns the username of the MySQL operator's management
// user.
func (i *Instance) GetUser() string {
	return "root"
}

// GetPassword returns the password of the MySQL operator's
// management user.
func (i *Instance) GetPassword() string {
	return os.Getenv("MYSQL_ROOT_PASSWORD")
}

// GetShellURI returns the MySQL shell URI for the local MySQL instance.
func (i *Instance) GetShellURI() string {
	return fmt.Sprintf("%s:%s@%s:%d", i.GetUser(), i.GetPassword(), i.Name(), i.Port)
}

// Name returns the name of the instance.
func (i *Instance) Name() string {
	return fmt.Sprintf("%s-%d", i.ParentName, i.Ordinal)
}

// statefulPodRegex is a regular expression that extracts the parent StatefulSet
// and ordinal from StatefulSet Pod's hostname.
var statefulPodRegex = regexp.MustCompile("(.*)-([0-9]+)$")

// getParentNameAndOrdinal gets the name of a Pod's parent StatefulSet and Pod's
// ordinal as extracted from its hostname. If the Pod was not created by a
// StatefulSet, its parent is considered to be empty string, and its ordinal is
// considered to be -1.
func getParentNameAndOrdinal(hostname string) (string, int) {
	parent := ""
	ordinal := -1
	subMatches := statefulPodRegex.FindStringSubmatch(hostname)
	if len(subMatches) < 3 {
		return parent, ordinal
	}
	parent = subMatches[1]
	if i, err := strconv.ParseInt(subMatches[2], 10, 32); err == nil {
		ordinal = int(i)
	}
	return parent, ordinal
}
