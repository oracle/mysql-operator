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

package manager

import (
	"context"
	"errors"
	"os"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	wait "k8s.io/apimachinery/pkg/util/wait"
	kubernetes "k8s.io/client-go/kubernetes"
	retry "k8s.io/client-go/util/retry"
	utilexec "k8s.io/utils/exec"

	"github.com/oracle/mysql-operator/pkg/cluster"
	"github.com/oracle/mysql-operator/pkg/cluster/innodb"
	"github.com/oracle/mysql-operator/pkg/util/mysqlsh"
)

var errNoClusterFound = errors.New("no cluster found on any of the seed nodes")

const defaultTimeout = 10 * time.Second

// isDatabaseRunning returns true if a connection can be made to the MySQL
// database running in the pod instance in which this function is called.
func isDatabaseRunning(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	err := utilexec.New().CommandContext(ctx,
		"mysqladmin",
		"--protocol", "tcp",
		"-u", "root",
		os.ExpandEnv("-p$MYSQL_ROOT_PASSWORD"),
		"status",
	).Run()
	return err == nil
}

func podExists(kubeclient kubernetes.Interface, instance *cluster.Instance) bool {
	err := wait.ExponentialBackoff(retry.DefaultRetry, func() (bool, error) {
		_, err := kubeclient.CoreV1().Pods(instance.Namespace).Get(instance.PodName(), metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	})
	return err == nil
}

// getReplicationGroupSeeds returns the list of servers in the replication
// group based on the given string (from the environment).
// It removes the local instance of mysql from the group.
func getReplicationGroupSeeds(seeds string, pod *cluster.Instance) ([]string, error) {
	s := []string{}
	for _, seed := range strings.Split(seeds, ",") {
		seedInstance, err := cluster.NewInstanceFromGroupSeed(seed)
		if err != nil {
			return nil, err
		}
		if seedInstance.Name() == pod.Name() {
			continue
		}
		s = append(s, seed)
	}
	return s, nil
}

// getClusterStatusFromGroupSeeds will attempt to get the cluster status (json)
// string for the MySQL cluster. It will try to log into the mysqlsh on each of
// the seed nodes in turn (excluding the current node) until it finds a valid
// cluster. If we can determine that no cluster is found on any of the seed
// nodes, then we return the empty string.
func getClusterStatusFromGroupSeeds(ctx context.Context, kubeclient kubernetes.Interface, pod *cluster.Instance) (*innodb.ClusterStatus, error) {
	replicationGroupSeeds, err := getReplicationGroupSeeds(os.Getenv("REPLICATION_GROUP_SEEDS"), pod)
	if err != nil {
		return nil, err
	}

	for i, replicationGroupSeed := range replicationGroupSeeds {
		inst, err := cluster.NewInstanceFromGroupSeed(replicationGroupSeed)
		if err != nil {
			return nil, err
		}
		if i == 0 || podExists(kubeclient, inst) {
			msh := mysqlsh.New(utilexec.New(), inst.GetShellURI())
			if !msh.IsClustered(ctx) {
				continue
			}
			return msh.GetClusterStatus(ctx)
		}
	}

	return nil, errNoClusterFound
}
