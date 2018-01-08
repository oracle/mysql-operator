package cluster

import (
	"fmt"

	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"

	"github.com/oracle/mysql-operator/pkg/constants"
	"github.com/oracle/mysql-operator/pkg/controllers/util"
	statefulsets "github.com/oracle/mysql-operator/pkg/resources/statefulsets"
)

// PodControlInterface defines the interface that the
// MySQLClusterController uses to create, update, and delete mysql pods. It
// is implemented as an interface to enable testing.
type PodControlInterface interface {
	PatchPod(old *v1.Pod, new *v1.Pod) error
}

type realPodControl struct {
	client    kubernetes.Interface
	podLister corelisters.PodLister
}

// NewRealPodControl creates a concrete implementation of the
// PodControlInterface.
func NewRealPodControl(client kubernetes.Interface, podLister corelisters.PodLister) PodControlInterface {
	return &realPodControl{client: client, podLister: podLister}
}

func (rpc *realPodControl) PatchPod(old *v1.Pod, new *v1.Pod) error {
	_, err := util.PatchPod(rpc.client, old, new)
	return err
}

// updatePodToOperatorVersion sets the specified MySQLOperator version on all valid parts
// of the Pod.
func updatePodToOperatorVersion(pod *v1.Pod, version string) *v1.Pod {
	newAgentImage := fmt.Sprintf("%s:%s", statefulsets.AgentImageName, version)

	pod.Labels[constants.MySQLOperatorVersionLabel] = version
	for idx, container := range pod.Spec.Containers {
		if container.Name == statefulsets.MySQLAgentContainerName {
			pod.Spec.Containers[idx].Image = newAgentImage
			break
		}
	}
	return pod
}
