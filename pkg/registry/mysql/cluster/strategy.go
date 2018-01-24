package cluster

import (
	"fmt"

	"github.com/golang/glog"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/names"

	"github.com/oracle/mysql-operator/pkg/apis/mysql"
)

type clusterStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

// implements interface RESTUpdateStrategy. This implementation validates updates to
// instance.Status updates only and disallows any modifications to the instance.Spec.
type clusterStatusStrategy struct {
	clusterStrategy
}

// newStrategy for MySQLClusters.
func newStrategy(typer runtime.ObjectTyper) (clusterStrategy, clusterStatusStrategy) {
	clusterStrategy := clusterStrategy{typer, names.SimpleNameGenerator}
	instanceStatusUpdateStrategy := clusterStatusStrategy{
		clusterStrategy,
	}
	return clusterStrategy, instanceStatusUpdateStrategy
}

// GetAttrs returns labels and fields of a given object for filtering purposes.
func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, bool, error) {
	instance, ok := obj.(*mysql.MySQLCluster)
	if !ok {
		return nil, nil, false, fmt.Errorf("given object is not a Instance")
	}
	return labels.Set(instance.ObjectMeta.Labels), ToSelectableFields(instance), instance.Initializers != nil, nil
}

// MatchInstance is the filter used by the generic etcd backend to watch events
// from etcd to clients of the apiserver only interested in specific labels/fields.
func MatchInstance(label labels.Selector, field fields.Selector) storage.SelectionPredicate {
	return storage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}

// ToSelectableFields returns a field set that represents the object.
func ToSelectableFields(obj *mysql.MySQLCluster) fields.Set {
	return generic.ObjectMetaFieldsSet(&obj.ObjectMeta, true)
}

func (clusterStrategy) NamespaceScoped() bool {
	return false
}

func (clusterStrategy) PrepareForCreate(ctx genericapirequest.Context, obj runtime.Object) {
	_, ok := obj.(*mysql.MySQLCluster)
	if !ok {
		glog.Fatal("received a non-instance object to create")
	}
}

func (clusterStrategy) PrepareForUpdate(ctx genericapirequest.Context, obj, old runtime.Object) {
	newInstance, ok := obj.(*mysql.MySQLCluster)
	if !ok {
		glog.Fatal("received a non-instance object to update to")
	}
	oldInstance, ok := old.(*mysql.MySQLCluster)
	if !ok {
		glog.Fatal("received a non-instance object to update from")
	}

	// Update should not change the status
	newInstance.Status = oldInstance.Status
}

func (clusterStrategy) Validate(ctx genericapirequest.Context, obj runtime.Object) field.ErrorList {
	return field.ErrorList{}
}

func (clusterStrategy) AllowCreateOnUpdate() bool {
	return false
}

func (clusterStrategy) AllowUnconditionalUpdate() bool {
	return false
}

func (clusterStrategy) Canonicalize(obj runtime.Object) {
}

func (clusterStrategy) ValidateUpdate(ctx genericapirequest.Context, obj, old runtime.Object) field.ErrorList {
	return field.ErrorList{}
}

func (clusterStatusStrategy) PrepareForUpdate(ctx genericapirequest.Context, new, old runtime.Object) {
	newCluster, ok := new.(*mysql.MySQLCluster)
	if !ok {
		glog.Fatal("received a non-cluster object to update to")
	}
	oldCluster, ok := old.(*mysql.MySQLCluster)
	if !ok {
		glog.Fatal("received a non-cluster object to update from")
	}
	// Status changes are not allowed to update spec
	newCluster.Spec = oldCluster.Spec
}

func (clusterStatusStrategy) ValidateUpdate(ctx genericapirequest.Context, new, old runtime.Object) field.ErrorList {
	return field.ErrorList{}
}
