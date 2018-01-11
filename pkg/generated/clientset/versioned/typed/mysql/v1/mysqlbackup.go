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

// MySQLBackupsGetter has a method to return a MySQLBackupInterface.
// A group's client should implement this interface.
type MySQLBackupsGetter interface {
	MySQLBackups(namespace string) MySQLBackupInterface
}

// MySQLBackupInterface has methods to work with MySQLBackup resources.
type MySQLBackupInterface interface {
	Create(*v1.MySQLBackup) (*v1.MySQLBackup, error)
	Update(*v1.MySQLBackup) (*v1.MySQLBackup, error)
	Delete(name string, options *meta_v1.DeleteOptions) error
	DeleteCollection(options *meta_v1.DeleteOptions, listOptions meta_v1.ListOptions) error
	Get(name string, options meta_v1.GetOptions) (*v1.MySQLBackup, error)
	List(opts meta_v1.ListOptions) (*v1.MySQLBackupList, error)
	Watch(opts meta_v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.MySQLBackup, err error)
	MySQLBackupExpansion
}

// mySQLBackups implements MySQLBackupInterface
type mySQLBackups struct {
	client rest.Interface
	ns     string
}

// newMySQLBackups returns a MySQLBackups
func newMySQLBackups(c *MysqlV1Client, namespace string) *mySQLBackups {
	return &mySQLBackups{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the mySQLBackup, and returns the corresponding mySQLBackup object, and an error if there is any.
func (c *mySQLBackups) Get(name string, options meta_v1.GetOptions) (result *v1.MySQLBackup, err error) {
	result = &v1.MySQLBackup{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("mysqlbackups").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of MySQLBackups that match those selectors.
func (c *mySQLBackups) List(opts meta_v1.ListOptions) (result *v1.MySQLBackupList, err error) {
	result = &v1.MySQLBackupList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("mysqlbackups").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested mySQLBackups.
func (c *mySQLBackups) Watch(opts meta_v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("mysqlbackups").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a mySQLBackup and creates it.  Returns the server's representation of the mySQLBackup, and an error, if there is any.
func (c *mySQLBackups) Create(mySQLBackup *v1.MySQLBackup) (result *v1.MySQLBackup, err error) {
	result = &v1.MySQLBackup{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("mysqlbackups").
		Body(mySQLBackup).
		Do().
		Into(result)
	return
}

// Update takes the representation of a mySQLBackup and updates it. Returns the server's representation of the mySQLBackup, and an error, if there is any.
func (c *mySQLBackups) Update(mySQLBackup *v1.MySQLBackup) (result *v1.MySQLBackup, err error) {
	result = &v1.MySQLBackup{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("mysqlbackups").
		Name(mySQLBackup.Name).
		Body(mySQLBackup).
		Do().
		Into(result)
	return
}

// Delete takes name of the mySQLBackup and deletes it. Returns an error if one occurs.
func (c *mySQLBackups) Delete(name string, options *meta_v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("mysqlbackups").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *mySQLBackups) DeleteCollection(options *meta_v1.DeleteOptions, listOptions meta_v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("mysqlbackups").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched mySQLBackup.
func (c *mySQLBackups) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.MySQLBackup, err error) {
	result = &v1.MySQLBackup{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("mysqlbackups").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
