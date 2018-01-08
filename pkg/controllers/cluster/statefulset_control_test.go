package cluster

import (
	"github.com/oracle/mysql-operator/pkg/controllers/util"
	apps "k8s.io/api/apps/v1beta1"
	kubernetes "k8s.io/client-go/kubernetes"
)

type fakeStatefulSetControl struct {
	StatefulSetControlInterface
	client kubernetes.Interface
}

// NewFakeStatefulSetControl creates a concrete FAKE implementation of the StatefulSetControlInterface.
func NewFakeStatefulSetControl(statefulSetControl StatefulSetControlInterface, client kubernetes.Interface) StatefulSetControlInterface {
	return &fakeStatefulSetControl{statefulSetControl, client}
}

func (rssc *fakeStatefulSetControl) PatchStatefulSet(old *apps.StatefulSet, new *apps.StatefulSet) error {
	_, err := util.UpdateStatefulSet(rssc.client, new)
	return err
}
