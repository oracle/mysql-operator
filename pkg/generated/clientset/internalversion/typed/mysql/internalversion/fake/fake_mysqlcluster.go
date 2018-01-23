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

package fake

import (
	mysql "github.com/oracle/mysql-operator/pkg/apis/mysql"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeMySQLClusters implements MySQLClusterInterface
type FakeMySQLClusters struct {
	Fake *FakeMysql
	ns   string
}

var mysqlclustersResource = schema.GroupVersionResource{Group: "mysql.oracle.com", Version: "", Resource: "mysqlclusters"}

var mysqlclustersKind = schema.GroupVersionKind{Group: "mysql.oracle.com", Version: "", Kind: "MySQLCluster"}

// Get takes name of the mySQLCluster, and returns the corresponding mySQLCluster object, and an error if there is any.
func (c *FakeMySQLClusters) Get(name string, options v1.GetOptions) (result *mysql.MySQLCluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(mysqlclustersResource, c.ns, name), &mysql.MySQLCluster{})

	if obj == nil {
		return nil, err
	}
	return obj.(*mysql.MySQLCluster), err
}

// List takes label and field selectors, and returns the list of MySQLClusters that match those selectors.
func (c *FakeMySQLClusters) List(opts v1.ListOptions) (result *mysql.MySQLClusterList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(mysqlclustersResource, mysqlclustersKind, c.ns, opts), &mysql.MySQLClusterList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &mysql.MySQLClusterList{}
	for _, item := range obj.(*mysql.MySQLClusterList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested mySQLClusters.
func (c *FakeMySQLClusters) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(mysqlclustersResource, c.ns, opts))

}

// Create takes the representation of a mySQLCluster and creates it.  Returns the server's representation of the mySQLCluster, and an error, if there is any.
func (c *FakeMySQLClusters) Create(mySQLCluster *mysql.MySQLCluster) (result *mysql.MySQLCluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(mysqlclustersResource, c.ns, mySQLCluster), &mysql.MySQLCluster{})

	if obj == nil {
		return nil, err
	}
	return obj.(*mysql.MySQLCluster), err
}

// Update takes the representation of a mySQLCluster and updates it. Returns the server's representation of the mySQLCluster, and an error, if there is any.
func (c *FakeMySQLClusters) Update(mySQLCluster *mysql.MySQLCluster) (result *mysql.MySQLCluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(mysqlclustersResource, c.ns, mySQLCluster), &mysql.MySQLCluster{})

	if obj == nil {
		return nil, err
	}
	return obj.(*mysql.MySQLCluster), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeMySQLClusters) UpdateStatus(mySQLCluster *mysql.MySQLCluster) (*mysql.MySQLCluster, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(mysqlclustersResource, "status", c.ns, mySQLCluster), &mysql.MySQLCluster{})

	if obj == nil {
		return nil, err
	}
	return obj.(*mysql.MySQLCluster), err
}

// Delete takes name of the mySQLCluster and deletes it. Returns an error if one occurs.
func (c *FakeMySQLClusters) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(mysqlclustersResource, c.ns, name), &mysql.MySQLCluster{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeMySQLClusters) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(mysqlclustersResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &mysql.MySQLClusterList{})
	return err
}

// Patch applies the patch and returns the patched mySQLCluster.
func (c *FakeMySQLClusters) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *mysql.MySQLCluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(mysqlclustersResource, c.ns, name, data, subresources...), &mysql.MySQLCluster{})

	if obj == nil {
		return nil, err
	}
	return obj.(*mysql.MySQLCluster), err
}
