package cluster

import (
	"fmt"

	apps "k8s.io/api/apps/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubernetes "k8s.io/client-go/kubernetes"
	appslisters "k8s.io/client-go/listers/apps/v1beta1"

	"github.com/oracle/mysql-operator/pkg/constants"
	"github.com/oracle/mysql-operator/pkg/controllers/util"
	statefulsets "github.com/oracle/mysql-operator/pkg/resources/statefulsets"
)

// StatefulSetControlInterface defines the interface that the
// MySQLClusterController uses to create, update, and delete StatefulSets. It
// is implemented as an interface to enable testing.
type StatefulSetControlInterface interface {
	CreateStatefulSet(ss *apps.StatefulSet) error
	DeleteStatefulSet(ss *apps.StatefulSet) error
	PatchStatefulSet(old *apps.StatefulSet, new *apps.StatefulSet) error
}

type realStatefulSetControl struct {
	client            kubernetes.Interface
	statefulSetLister appslisters.StatefulSetLister
}

// NewRealStatefulSetControl creates a concrete implementation of the
// StatefulSetControlInterface.
func NewRealStatefulSetControl(client kubernetes.Interface, statefulSetLister appslisters.StatefulSetLister) StatefulSetControlInterface {
	return &realStatefulSetControl{client: client, statefulSetLister: statefulSetLister}
}

func (rssc *realStatefulSetControl) CreateStatefulSet(ss *apps.StatefulSet) error {
	_, err := rssc.client.AppsV1beta1().StatefulSets(ss.Namespace).Create(ss)
	return err
}

func (rssc *realStatefulSetControl) DeleteStatefulSet(ss *apps.StatefulSet) error {
	policy := metav1.DeletePropagationBackground
	opts := &metav1.DeleteOptions{PropagationPolicy: &policy}
	err := rssc.client.AppsV1beta1().StatefulSets(ss.Namespace).Delete(ss.Name, opts)
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

func (rssc *realStatefulSetControl) PatchStatefulSet(old *apps.StatefulSet, new *apps.StatefulSet) error {
	_, err := util.PatchStatefulSet(rssc.client, old, new)
	return err
}

// updateStatefulSetToOperatorVersion sets the specified MySQLOperator version on all valid parts
// of the StatefulSet.
func updateStatefulSetToOperatorVersion(ss *apps.StatefulSet, version string) *apps.StatefulSet {
	newAgentImage := fmt.Sprintf("%s:%s", statefulsets.AgentImageName, version)

	ss.ObjectMeta.Labels[constants.MySQLOperatorVersionLabel] = version
	for idx, container := range ss.Spec.Template.Spec.Containers {
		if container.Name == statefulsets.MySQLAgentContainerName {
			ss.Spec.Template.Spec.Containers[idx].Image = newAgentImage
			break
		}
	}

	return ss
}
