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

package internalversion

import (
	mysql "github.com/oracle/mysql-operator/pkg/apis/mysql"
	scheme "github.com/oracle/mysql-operator/pkg/generated/clientset/internalversion/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// MySQLRestoresGetter has a method to return a MySQLRestoreInterface.
// A group's client should implement this interface.
type MySQLRestoresGetter interface {
	MySQLRestores(namespace string) MySQLRestoreInterface
}

// MySQLRestoreInterface has methods to work with MySQLRestore resources.
type MySQLRestoreInterface interface {
	Create(*mysql.MySQLRestore) (*mysql.MySQLRestore, error)
	Update(*mysql.MySQLRestore) (*mysql.MySQLRestore, error)
	UpdateStatus(*mysql.MySQLRestore) (*mysql.MySQLRestore, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*mysql.MySQLRestore, error)
	List(opts v1.ListOptions) (*mysql.MySQLRestoreList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *mysql.MySQLRestore, err error)
	MySQLRestoreExpansion
}

// mySQLRestores implements MySQLRestoreInterface
type mySQLRestores struct {
	client rest.Interface
	ns     string
}

// newMySQLRestores returns a MySQLRestores
func newMySQLRestores(c *MysqlClient, namespace string) *mySQLRestores {
	return &mySQLRestores{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the mySQLRestore, and returns the corresponding mySQLRestore object, and an error if there is any.
func (c *mySQLRestores) Get(name string, options v1.GetOptions) (result *mysql.MySQLRestore, err error) {
	result = &mysql.MySQLRestore{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("mysqlrestores").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of MySQLRestores that match those selectors.
func (c *mySQLRestores) List(opts v1.ListOptions) (result *mysql.MySQLRestoreList, err error) {
	result = &mysql.MySQLRestoreList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("mysqlrestores").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested mySQLRestores.
func (c *mySQLRestores) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("mysqlrestores").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a mySQLRestore and creates it.  Returns the server's representation of the mySQLRestore, and an error, if there is any.
func (c *mySQLRestores) Create(mySQLRestore *mysql.MySQLRestore) (result *mysql.MySQLRestore, err error) {
	result = &mysql.MySQLRestore{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("mysqlrestores").
		Body(mySQLRestore).
		Do().
		Into(result)
	return
}

// Update takes the representation of a mySQLRestore and updates it. Returns the server's representation of the mySQLRestore, and an error, if there is any.
func (c *mySQLRestores) Update(mySQLRestore *mysql.MySQLRestore) (result *mysql.MySQLRestore, err error) {
	result = &mysql.MySQLRestore{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("mysqlrestores").
		Name(mySQLRestore.Name).
		Body(mySQLRestore).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *mySQLRestores) UpdateStatus(mySQLRestore *mysql.MySQLRestore) (result *mysql.MySQLRestore, err error) {
	result = &mysql.MySQLRestore{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("mysqlrestores").
		Name(mySQLRestore.Name).
		SubResource("status").
		Body(mySQLRestore).
		Do().
		Into(result)
	return
}

// Delete takes name of the mySQLRestore and deletes it. Returns an error if one occurs.
func (c *mySQLRestores) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("mysqlrestores").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *mySQLRestores) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("mysqlrestores").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched mySQLRestore.
func (c *mySQLRestores) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *mysql.MySQLRestore, err error) {
	result = &mysql.MySQLRestore{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("mysqlrestores").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
