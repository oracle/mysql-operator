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

// MySQLClustersGetter has a method to return a MySQLClusterInterface.
// A group's client should implement this interface.
type MySQLClustersGetter interface {
	MySQLClusters(namespace string) MySQLClusterInterface
}

// MySQLClusterInterface has methods to work with MySQLCluster resources.
type MySQLClusterInterface interface {
	Create(*v1.MySQLCluster) (*v1.MySQLCluster, error)
	Update(*v1.MySQLCluster) (*v1.MySQLCluster, error)
	Delete(name string, options *meta_v1.DeleteOptions) error
	DeleteCollection(options *meta_v1.DeleteOptions, listOptions meta_v1.ListOptions) error
	Get(name string, options meta_v1.GetOptions) (*v1.MySQLCluster, error)
	List(opts meta_v1.ListOptions) (*v1.MySQLClusterList, error)
	Watch(opts meta_v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.MySQLCluster, err error)
	MySQLClusterExpansion
}

// mySQLClusters implements MySQLClusterInterface
type mySQLClusters struct {
	client rest.Interface
	ns     string
}

// newMySQLClusters returns a MySQLClusters
func newMySQLClusters(c *MysqlV1Client, namespace string) *mySQLClusters {
	return &mySQLClusters{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the mySQLCluster, and returns the corresponding mySQLCluster object, and an error if there is any.
func (c *mySQLClusters) Get(name string, options meta_v1.GetOptions) (result *v1.MySQLCluster, err error) {
	result = &v1.MySQLCluster{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("mysqlclusters").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of MySQLClusters that match those selectors.
func (c *mySQLClusters) List(opts meta_v1.ListOptions) (result *v1.MySQLClusterList, err error) {
	result = &v1.MySQLClusterList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("mysqlclusters").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested mySQLClusters.
func (c *mySQLClusters) Watch(opts meta_v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("mysqlclusters").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a mySQLCluster and creates it.  Returns the server's representation of the mySQLCluster, and an error, if there is any.
func (c *mySQLClusters) Create(mySQLCluster *v1.MySQLCluster) (result *v1.MySQLCluster, err error) {
	result = &v1.MySQLCluster{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("mysqlclusters").
		Body(mySQLCluster).
		Do().
		Into(result)
	return
}

// Update takes the representation of a mySQLCluster and updates it. Returns the server's representation of the mySQLCluster, and an error, if there is any.
func (c *mySQLClusters) Update(mySQLCluster *v1.MySQLCluster) (result *v1.MySQLCluster, err error) {
	result = &v1.MySQLCluster{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("mysqlclusters").
		Name(mySQLCluster.Name).
		Body(mySQLCluster).
		Do().
		Into(result)
	return
}

// Delete takes name of the mySQLCluster and deletes it. Returns an error if one occurs.
func (c *mySQLClusters) Delete(name string, options *meta_v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("mysqlclusters").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *mySQLClusters) DeleteCollection(options *meta_v1.DeleteOptions, listOptions meta_v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("mysqlclusters").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched mySQLCluster.
func (c *mySQLClusters) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.MySQLCluster, err error) {
	result = &v1.MySQLCluster{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("mysqlclusters").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
