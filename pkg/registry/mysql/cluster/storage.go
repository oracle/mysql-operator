package cluster

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	generic "k8s.io/apiserver/pkg/registry/generic"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
	rest "k8s.io/apiserver/pkg/registry/rest"

	"github.com/oracle/mysql-operator/pkg/apis/mysql"
	"github.com/oracle/mysql-operator/pkg/registry"
)

// NewREST returns a RESTStorage object that will work against API services.
func NewREST(scheme *runtime.Scheme, optsGetter generic.RESTOptionsGetter) (rest.Storage, rest.Storage, error) {
	strategy, statusStrategy := newStrategy(scheme)

	store := genericregistry.Store{
		NewFunc:                  func() runtime.Object { return &mysql.MySQLCluster{} },
		NewListFunc:              func() runtime.Object { return &mysql.MySQLClusterList{} },
		PredicateFunc:            MatchInstance,
		DefaultQualifiedResource: mysql.Resource("mysqlclusters"),

		CreateStrategy: strategy,
		UpdateStrategy: strategy,
		DeleteStrategy: strategy,
	}

	options := &generic.StoreOptions{RESTOptions: optsGetter, AttrFunc: GetAttrs}
	if err := store.CompleteWithOptions(options); err != nil {
		return nil, nil, err
	}

	statusStore := store
	statusStore.UpdateStrategy = statusStrategy

	return &registry.REST{Store: &store}, &StatusREST{store: &statusStore}, nil
}

// StatusREST defines the REST operations for the status subresource via
// implementation of various rest interfaces.  It supports the http verbs GET,
// PATCH, and PUT.
type StatusREST struct {
	store *genericregistry.Store
}

// New returns a new ServiceClass
func (r *StatusREST) New() runtime.Object {
	return &mysql.MySQLCluster{}
}

// Get retrieves the object from the storage. It is required to support Patch
// and to implement the rest.Getter interface.
func (r *StatusREST) Get(ctx genericapirequest.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	return r.store.Get(ctx, name, options)
}

// Update alters the status subset of an object and it
// implements rest.Updater interface
func (r *StatusREST) Update(ctx genericapirequest.Context, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc) (runtime.Object, bool, error) {
	return r.store.Update(ctx, name, objInfo, createValidation, updateValidation)
}
