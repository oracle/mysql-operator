// Copyright 2018 Oracle and/or its affiliates. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cluster

import (
	"fmt"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/oracle/mysql-operator/pkg/cluster/innodb"
)

// Instance represents the local MySQL instance.
type Instance struct {
	// Namespace is the Kubernetes Namespace in which the instance is running.
	Namespace string
	// ClusterName is the name of the Cluster to which the instance
	// belongs.
	ClusterName string
	// ParentName is the name of the StatefulSet to which the instance belongs.
	ParentName string
	// Ordinal is the StatefulSet ordinal of the instances Pod.
	Ordinal int
	// Port is the port on which MySQLDB is listening.
	Port int
	// MultiMaster specifies if all, or just a single, instance is configured to be read/write.
	MultiMaster bool

	// IP is the IP address of the Kubernetes Pod.
	IP net.IP
}

// NewInstance creates a new Instance.
func NewInstance(namespace, clusterName, parentName string, ordinal, port int, multiMaster bool) *Instance {
	return &Instance{
		Namespace:   namespace,
		ClusterName: clusterName,
		ParentName:  parentName,
		Ordinal:     ordinal,
		Port:        port,
		MultiMaster: multiMaster,
	}
}

// NewLocalInstance creates a new instance of this structure, with it's name and index
// populated from os.Hostname().
func NewLocalInstance() (*Instance, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	name, ordinal := GetParentNameAndOrdinal(hostname)
	multiMaster, _ := strconv.ParseBool(os.Getenv("MYSQL_CLUSTER_MULTI_MASTER"))
	return &Instance{
		Namespace:   os.Getenv("POD_NAMESPACE"),
		ClusterName: os.Getenv("MYSQL_CLUSTER_NAME"),
		ParentName:  name,
		Ordinal:     ordinal,
		Port:        innodb.MySQLDBPort,
		MultiMaster: multiMaster,
		IP:          net.ParseIP(os.Getenv("MY_POD_IP")),
	}, nil
}

// NewInstanceFromGroupSeed creates an Instance from a fully qualified group
// seed.
func NewInstanceFromGroupSeed(seed string) (*Instance, error) {
	podName, err := podNameFromSeed(seed)
	if err != nil {
		return nil, errors.Wrap(err, "getting pod name from group seed")
	}
	// We don't care about the returned port here as the Instance's port its
	// MySQLDB port not its group replication port.
	parentName, ordinal := GetParentNameAndOrdinal(podName)
	multiMaster, _ := strconv.ParseBool(os.Getenv("MYSQL_CLUSTER_MULTI_MASTER"))
	return &Instance{
		ClusterName: os.Getenv("MYSQL_CLUSTER_NAME"),
		Namespace:   os.Getenv("POD_NAMESPACE"),
		ParentName:  parentName,
		Ordinal:     ordinal,
		Port:        innodb.MySQLDBPort,
		MultiMaster: multiMaster,
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
	return fmt.Sprintf("%s.%s", i.PodName(), i.ParentName)
}

// PodName returns the name of the instance's Pod.
func (i *Instance) PodName() string {
	return fmt.Sprintf("%s-%d", i.ParentName, i.Ordinal)
}

// WhitelistCIDR returns the CIDR range to whitelist for GR based on the Pod's IP.
func (i *Instance) WhitelistCIDR() (string, error) {
	var privateRanges []*net.IPNet

	for _, addrRange := range []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"100.64.0.0/10", // IPv4 shared address space (RFC 6598), improperly used by kops
	} {
		_, block, _ := net.ParseCIDR(addrRange)
		privateRanges = append(privateRanges, block)
	}

	for _, block := range privateRanges {
		if block.Contains(i.IP) {
			return block.String(), nil
		}
	}

	return "", errors.Errorf("pod IP %q is not a private IPv4 address", i.IP.String())
}

// statefulPodRegex is a regular expression that extracts the parent StatefulSet
// and ordinal from StatefulSet Pod's hostname.
var statefulPodRegex = regexp.MustCompile("(.*)-([0-9]+)$")

// GetParentNameAndOrdinal gets the name of a Pod's parent StatefulSet and Pod's
// ordinal from the Pods name (or hostname). If the Pod was not created by a
// StatefulSet, its parent is considered to be empty string, and its ordinal is
// considered to be -1.
func GetParentNameAndOrdinal(name string) (string, int) {
	parent := ""
	ordinal := -1
	subMatches := statefulPodRegex.FindStringSubmatch(name)
	if len(subMatches) < 3 {
		return parent, ordinal
	}
	parent = subMatches[1]
	if i, err := strconv.ParseInt(subMatches[2], 10, 32); err == nil {
		ordinal = int(i)
	}
	return parent, ordinal
}

func podNameFromSeed(seed string) (string, error) {
	host, _, err := net.SplitHostPort(seed)
	if err != nil {
		return "", errors.Wrap(err, "splitting host and port")
	}
	return strings.SplitN(host, ".", 2)[0], nil
}
