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
	mysql_v1 "github.com/oracle/mysql-operator/pkg/apis/mysql/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeMySQLRestores implements MySQLRestoreInterface
type FakeMySQLRestores struct {
	Fake *FakeMysqlV1
	ns   string
}

var mysqlrestoresResource = schema.GroupVersionResource{Group: "mysql.oracle.com", Version: "v1", Resource: "mysqlrestores"}

var mysqlrestoresKind = schema.GroupVersionKind{Group: "mysql.oracle.com", Version: "v1", Kind: "MySQLRestore"}

// Get takes name of the mySQLRestore, and returns the corresponding mySQLRestore object, and an error if there is any.
func (c *FakeMySQLRestores) Get(name string, options v1.GetOptions) (result *mysql_v1.MySQLRestore, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(mysqlrestoresResource, c.ns, name), &mysql_v1.MySQLRestore{})

	if obj == nil {
		return nil, err
	}
	return obj.(*mysql_v1.MySQLRestore), err
}

// List takes label and field selectors, and returns the list of MySQLRestores that match those selectors.
func (c *FakeMySQLRestores) List(opts v1.ListOptions) (result *mysql_v1.MySQLRestoreList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(mysqlrestoresResource, mysqlrestoresKind, c.ns, opts), &mysql_v1.MySQLRestoreList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &mysql_v1.MySQLRestoreList{}
	for _, item := range obj.(*mysql_v1.MySQLRestoreList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested mySQLRestores.
func (c *FakeMySQLRestores) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(mysqlrestoresResource, c.ns, opts))

}

// Create takes the representation of a mySQLRestore and creates it.  Returns the server's representation of the mySQLRestore, and an error, if there is any.
func (c *FakeMySQLRestores) Create(mySQLRestore *mysql_v1.MySQLRestore) (result *mysql_v1.MySQLRestore, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(mysqlrestoresResource, c.ns, mySQLRestore), &mysql_v1.MySQLRestore{})

	if obj == nil {
		return nil, err
	}
	return obj.(*mysql_v1.MySQLRestore), err
}

// Update takes the representation of a mySQLRestore and updates it. Returns the server's representation of the mySQLRestore, and an error, if there is any.
func (c *FakeMySQLRestores) Update(mySQLRestore *mysql_v1.MySQLRestore) (result *mysql_v1.MySQLRestore, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(mysqlrestoresResource, c.ns, mySQLRestore), &mysql_v1.MySQLRestore{})

	if obj == nil {
		return nil, err
	}
	return obj.(*mysql_v1.MySQLRestore), err
}

// Delete takes name of the mySQLRestore and deletes it. Returns an error if one occurs.
func (c *FakeMySQLRestores) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(mysqlrestoresResource, c.ns, name), &mysql_v1.MySQLRestore{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeMySQLRestores) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(mysqlrestoresResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &mysql_v1.MySQLRestoreList{})
	return err
}

// Patch applies the patch and returns the patched mySQLRestore.
func (c *FakeMySQLRestores) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *mysql_v1.MySQLRestore, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(mysqlrestoresResource, c.ns, name, data, subresources...), &mysql_v1.MySQLRestore{})

	if obj == nil {
		return nil, err
	}
	return obj.(*mysql_v1.MySQLRestore), err
}
