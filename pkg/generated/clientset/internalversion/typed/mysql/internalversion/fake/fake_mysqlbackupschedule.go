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

// FakeMySQLBackupSchedules implements MySQLBackupScheduleInterface
type FakeMySQLBackupSchedules struct {
	Fake *FakeMysql
	ns   string
}

var mysqlbackupschedulesResource = schema.GroupVersionResource{Group: "mysql.oracle.com", Version: "", Resource: "mysqlbackupschedules"}

var mysqlbackupschedulesKind = schema.GroupVersionKind{Group: "mysql.oracle.com", Version: "", Kind: "MySQLBackupSchedule"}

// Get takes name of the mySQLBackupSchedule, and returns the corresponding mySQLBackupSchedule object, and an error if there is any.
func (c *FakeMySQLBackupSchedules) Get(name string, options v1.GetOptions) (result *mysql.MySQLBackupSchedule, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(mysqlbackupschedulesResource, c.ns, name), &mysql.MySQLBackupSchedule{})

	if obj == nil {
		return nil, err
	}
	return obj.(*mysql.MySQLBackupSchedule), err
}

// List takes label and field selectors, and returns the list of MySQLBackupSchedules that match those selectors.
func (c *FakeMySQLBackupSchedules) List(opts v1.ListOptions) (result *mysql.MySQLBackupScheduleList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(mysqlbackupschedulesResource, mysqlbackupschedulesKind, c.ns, opts), &mysql.MySQLBackupScheduleList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &mysql.MySQLBackupScheduleList{}
	for _, item := range obj.(*mysql.MySQLBackupScheduleList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested mySQLBackupSchedules.
func (c *FakeMySQLBackupSchedules) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(mysqlbackupschedulesResource, c.ns, opts))

}

// Create takes the representation of a mySQLBackupSchedule and creates it.  Returns the server's representation of the mySQLBackupSchedule, and an error, if there is any.
func (c *FakeMySQLBackupSchedules) Create(mySQLBackupSchedule *mysql.MySQLBackupSchedule) (result *mysql.MySQLBackupSchedule, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(mysqlbackupschedulesResource, c.ns, mySQLBackupSchedule), &mysql.MySQLBackupSchedule{})

	if obj == nil {
		return nil, err
	}
	return obj.(*mysql.MySQLBackupSchedule), err
}

// Update takes the representation of a mySQLBackupSchedule and updates it. Returns the server's representation of the mySQLBackupSchedule, and an error, if there is any.
func (c *FakeMySQLBackupSchedules) Update(mySQLBackupSchedule *mysql.MySQLBackupSchedule) (result *mysql.MySQLBackupSchedule, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(mysqlbackupschedulesResource, c.ns, mySQLBackupSchedule), &mysql.MySQLBackupSchedule{})

	if obj == nil {
		return nil, err
	}
	return obj.(*mysql.MySQLBackupSchedule), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeMySQLBackupSchedules) UpdateStatus(mySQLBackupSchedule *mysql.MySQLBackupSchedule) (*mysql.MySQLBackupSchedule, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(mysqlbackupschedulesResource, "status", c.ns, mySQLBackupSchedule), &mysql.MySQLBackupSchedule{})

	if obj == nil {
		return nil, err
	}
	return obj.(*mysql.MySQLBackupSchedule), err
}

// Delete takes name of the mySQLBackupSchedule and deletes it. Returns an error if one occurs.
func (c *FakeMySQLBackupSchedules) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(mysqlbackupschedulesResource, c.ns, name), &mysql.MySQLBackupSchedule{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeMySQLBackupSchedules) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(mysqlbackupschedulesResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &mysql.MySQLBackupScheduleList{})
	return err
}

// Patch applies the patch and returns the patched mySQLBackupSchedule.
func (c *FakeMySQLBackupSchedules) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *mysql.MySQLBackupSchedule, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(mysqlbackupschedulesResource, c.ns, name, data, subresources...), &mysql.MySQLBackupSchedule{})

	if obj == nil {
		return nil, err
	}
	return obj.(*mysql.MySQLBackupSchedule), err
}
