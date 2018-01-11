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

package v1

import (
	v1 "github.com/oracle/mysql-operator/pkg/apis/mysql/v1"
	scheme "github.com/oracle/mysql-operator/pkg/generated/clientset/versioned/scheme"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// MySQLBackupSchedulesGetter has a method to return a MySQLBackupScheduleInterface.
// A group's client should implement this interface.
type MySQLBackupSchedulesGetter interface {
	MySQLBackupSchedules(namespace string) MySQLBackupScheduleInterface
}

// MySQLBackupScheduleInterface has methods to work with MySQLBackupSchedule resources.
type MySQLBackupScheduleInterface interface {
	Create(*v1.MySQLBackupSchedule) (*v1.MySQLBackupSchedule, error)
	Update(*v1.MySQLBackupSchedule) (*v1.MySQLBackupSchedule, error)
	Delete(name string, options *meta_v1.DeleteOptions) error
	DeleteCollection(options *meta_v1.DeleteOptions, listOptions meta_v1.ListOptions) error
	Get(name string, options meta_v1.GetOptions) (*v1.MySQLBackupSchedule, error)
	List(opts meta_v1.ListOptions) (*v1.MySQLBackupScheduleList, error)
	Watch(opts meta_v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.MySQLBackupSchedule, err error)
	MySQLBackupScheduleExpansion
}

// mySQLBackupSchedules implements MySQLBackupScheduleInterface
type mySQLBackupSchedules struct {
	client rest.Interface
	ns     string
}

// newMySQLBackupSchedules returns a MySQLBackupSchedules
func newMySQLBackupSchedules(c *MysqlV1Client, namespace string) *mySQLBackupSchedules {
	return &mySQLBackupSchedules{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the mySQLBackupSchedule, and returns the corresponding mySQLBackupSchedule object, and an error if there is any.
func (c *mySQLBackupSchedules) Get(name string, options meta_v1.GetOptions) (result *v1.MySQLBackupSchedule, err error) {
	result = &v1.MySQLBackupSchedule{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("mysqlbackupschedules").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of MySQLBackupSchedules that match those selectors.
func (c *mySQLBackupSchedules) List(opts meta_v1.ListOptions) (result *v1.MySQLBackupScheduleList, err error) {
	result = &v1.MySQLBackupScheduleList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("mysqlbackupschedules").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested mySQLBackupSchedules.
func (c *mySQLBackupSchedules) Watch(opts meta_v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("mysqlbackupschedules").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a mySQLBackupSchedule and creates it.  Returns the server's representation of the mySQLBackupSchedule, and an error, if there is any.
func (c *mySQLBackupSchedules) Create(mySQLBackupSchedule *v1.MySQLBackupSchedule) (result *v1.MySQLBackupSchedule, err error) {
	result = &v1.MySQLBackupSchedule{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("mysqlbackupschedules").
		Body(mySQLBackupSchedule).
		Do().
		Into(result)
	return
}

// Update takes the representation of a mySQLBackupSchedule and updates it. Returns the server's representation of the mySQLBackupSchedule, and an error, if there is any.
func (c *mySQLBackupSchedules) Update(mySQLBackupSchedule *v1.MySQLBackupSchedule) (result *v1.MySQLBackupSchedule, err error) {
	result = &v1.MySQLBackupSchedule{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("mysqlbackupschedules").
		Name(mySQLBackupSchedule.Name).
		Body(mySQLBackupSchedule).
		Do().
		Into(result)
	return
}

// Delete takes name of the mySQLBackupSchedule and deletes it. Returns an error if one occurs.
func (c *mySQLBackupSchedules) Delete(name string, options *meta_v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("mysqlbackupschedules").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *mySQLBackupSchedules) DeleteCollection(options *meta_v1.DeleteOptions, listOptions meta_v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("mysqlbackupschedules").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched mySQLBackupSchedule.
func (c *mySQLBackupSchedules) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.MySQLBackupSchedule, err error) {
	result = &v1.MySQLBackupSchedule{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("mysqlbackupschedules").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
