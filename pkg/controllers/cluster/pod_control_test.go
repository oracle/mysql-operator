package cluster

import (
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/oracle/mysql-operator/pkg/controllers/util"
)

type fakePodControl struct {
	PodControlInterface
	client kubernetes.Interface
}

// NewFakePodControl creates a concrete FAKE implementation of the PodControlInterface.
func NewFakePodControl(podControl PodControlInterface, client kubernetes.Interface) PodControlInterface {
	return &fakePodControl{podControl, client}
}

func (rpc *fakePodControl) PatchPod(old *v1.Pod, new *v1.Pod) error {
	_, err := util.UpdatePod(rpc.client, new)
	return err
}
