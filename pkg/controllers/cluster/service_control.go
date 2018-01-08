package cluster

import (
	"k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
)

// ServiceControlInterface defines the interface that the
// MySQLClusterController uses to create, update, and delete Services. It
// is implemented as an interface to enable testing.
type ServiceControlInterface interface {
	CreateService(s *v1.Service) error
	DeleteService(s *v1.Service) error
}

type realServiceControl struct {
	client        kubernetes.Interface
	serviceLister corelisters.ServiceLister
}

// NewRealServiceControl creates a concrete implementation of the
// ServiceControlInterface.
func NewRealServiceControl(client kubernetes.Interface, serviceLister corelisters.ServiceLister) ServiceControlInterface {
	return &realServiceControl{client: client, serviceLister: serviceLister}
}

func (rsc *realServiceControl) CreateService(s *v1.Service) error {
	_, err := rsc.client.CoreV1().Services(s.Namespace).Create(s)
	return err
}

func (rsc *realServiceControl) DeleteService(s *v1.Service) error {
	err := rsc.client.CoreV1().Services(s.Namespace).Delete(s.Name, nil)
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}
