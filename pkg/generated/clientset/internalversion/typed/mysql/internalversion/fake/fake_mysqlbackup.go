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

// FakeMySQLBackups implements MySQLBackupInterface
type FakeMySQLBackups struct {
	Fake *FakeMysql
	ns   string
}

var mysqlbackupsResource = schema.GroupVersionResource{Group: "mysql.oracle.com", Version: "", Resource: "mysqlbackups"}

var mysqlbackupsKind = schema.GroupVersionKind{Group: "mysql.oracle.com", Version: "", Kind: "MySQLBackup"}

// Get takes name of the mySQLBackup, and returns the corresponding mySQLBackup object, and an error if there is any.
func (c *FakeMySQLBackups) Get(name string, options v1.GetOptions) (result *mysql.MySQLBackup, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(mysqlbackupsResource, c.ns, name), &mysql.MySQLBackup{})

	if obj == nil {
		return nil, err
	}
	return obj.(*mysql.MySQLBackup), err
}

// List takes label and field selectors, and returns the list of MySQLBackups that match those selectors.
func (c *FakeMySQLBackups) List(opts v1.ListOptions) (result *mysql.MySQLBackupList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(mysqlbackupsResource, mysqlbackupsKind, c.ns, opts), &mysql.MySQLBackupList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &mysql.MySQLBackupList{}
	for _, item := range obj.(*mysql.MySQLBackupList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested mySQLBackups.
func (c *FakeMySQLBackups) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(mysqlbackupsResource, c.ns, opts))

}

// Create takes the representation of a mySQLBackup and creates it.  Returns the server's representation of the mySQLBackup, and an error, if there is any.
func (c *FakeMySQLBackups) Create(mySQLBackup *mysql.MySQLBackup) (result *mysql.MySQLBackup, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(mysqlbackupsResource, c.ns, mySQLBackup), &mysql.MySQLBackup{})

	if obj == nil {
		return nil, err
	}
	return obj.(*mysql.MySQLBackup), err
}

// Update takes the representation of a mySQLBackup and updates it. Returns the server's representation of the mySQLBackup, and an error, if there is any.
func (c *FakeMySQLBackups) Update(mySQLBackup *mysql.MySQLBackup) (result *mysql.MySQLBackup, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(mysqlbackupsResource, c.ns, mySQLBackup), &mysql.MySQLBackup{})

	if obj == nil {
		return nil, err
	}
	return obj.(*mysql.MySQLBackup), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeMySQLBackups) UpdateStatus(mySQLBackup *mysql.MySQLBackup) (*mysql.MySQLBackup, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(mysqlbackupsResource, "status", c.ns, mySQLBackup), &mysql.MySQLBackup{})

	if obj == nil {
		return nil, err
	}
	return obj.(*mysql.MySQLBackup), err
}

// Delete takes name of the mySQLBackup and deletes it. Returns an error if one occurs.
func (c *FakeMySQLBackups) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(mysqlbackupsResource, c.ns, name), &mysql.MySQLBackup{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeMySQLBackups) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(mysqlbackupsResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &mysql.MySQLBackupList{})
	return err
}

// Patch applies the patch and returns the patched mySQLBackup.
func (c *FakeMySQLBackups) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *mysql.MySQLBackup, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(mysqlbackupsResource, c.ns, name, data, subresources...), &mysql.MySQLBackup{})

	if obj == nil {
		return nil, err
	}
	return obj.(*mysql.MySQLBackup), err
}
