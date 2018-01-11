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

// MySQLRestoresGetter has a method to return a MySQLRestoreInterface.
// A group's client should implement this interface.
type MySQLRestoresGetter interface {
	MySQLRestores(namespace string) MySQLRestoreInterface
}

// MySQLRestoreInterface has methods to work with MySQLRestore resources.
type MySQLRestoreInterface interface {
	Create(*v1.MySQLRestore) (*v1.MySQLRestore, error)
	Update(*v1.MySQLRestore) (*v1.MySQLRestore, error)
	Delete(name string, options *meta_v1.DeleteOptions) error
	DeleteCollection(options *meta_v1.DeleteOptions, listOptions meta_v1.ListOptions) error
	Get(name string, options meta_v1.GetOptions) (*v1.MySQLRestore, error)
	List(opts meta_v1.ListOptions) (*v1.MySQLRestoreList, error)
	Watch(opts meta_v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.MySQLRestore, err error)
	MySQLRestoreExpansion
}

// mySQLRestores implements MySQLRestoreInterface
type mySQLRestores struct {
	client rest.Interface
	ns     string
}

// newMySQLRestores returns a MySQLRestores
func newMySQLRestores(c *MysqlV1Client, namespace string) *mySQLRestores {
	return &mySQLRestores{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the mySQLRestore, and returns the corresponding mySQLRestore object, and an error if there is any.
func (c *mySQLRestores) Get(name string, options meta_v1.GetOptions) (result *v1.MySQLRestore, err error) {
	result = &v1.MySQLRestore{}
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
func (c *mySQLRestores) List(opts meta_v1.ListOptions) (result *v1.MySQLRestoreList, err error) {
	result = &v1.MySQLRestoreList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("mysqlrestores").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested mySQLRestores.
func (c *mySQLRestores) Watch(opts meta_v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("mysqlrestores").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a mySQLRestore and creates it.  Returns the server's representation of the mySQLRestore, and an error, if there is any.
func (c *mySQLRestores) Create(mySQLRestore *v1.MySQLRestore) (result *v1.MySQLRestore, err error) {
	result = &v1.MySQLRestore{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("mysqlrestores").
		Body(mySQLRestore).
		Do().
		Into(result)
	return
}

// Update takes the representation of a mySQLRestore and updates it. Returns the server's representation of the mySQLRestore, and an error, if there is any.
func (c *mySQLRestores) Update(mySQLRestore *v1.MySQLRestore) (result *v1.MySQLRestore, err error) {
	result = &v1.MySQLRestore{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("mysqlrestores").
		Name(mySQLRestore.Name).
		Body(mySQLRestore).
		Do().
		Into(result)
	return
}

// Delete takes name of the mySQLRestore and deletes it. Returns an error if one occurs.
func (c *mySQLRestores) Delete(name string, options *meta_v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("mysqlrestores").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *mySQLRestores) DeleteCollection(options *meta_v1.DeleteOptions, listOptions meta_v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("mysqlrestores").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched mySQLRestore.
func (c *mySQLRestores) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.MySQLRestore, err error) {
	result = &v1.MySQLRestore{}
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
