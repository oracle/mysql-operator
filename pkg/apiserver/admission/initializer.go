package admission

import (
	"k8s.io/apiserver/pkg/admission"

	clientset "github.com/oracle/mysql-operator/pkg/generated/clientset/internalversion"
	informers "github.com/oracle/mysql-operator/pkg/generated/informers/internalversion"
)

// WantsInternalMySQLClientSet defines a function which sets ClientSet for admission plugins that need it
type WantsInternalMySQLClientSet interface {
	SetInternalMySQLClientSet(clientset.Interface)
	admission.InitializationValidator
}

// WantsInternalMySQLInformerFactory defines a function which sets InformerFactory for admission plugins that need it
type WantsInternalMySQLInformerFactory interface {
	SetInternalMySQLInformerFactory(informers.SharedInformerFactory)
	admission.InitializationValidator
}

type pluginInitializer struct {
	internalClient clientset.Interface
	informers      informers.SharedInformerFactory
}

var _ admission.PluginInitializer = pluginInitializer{}

// NewPluginInitializer constructs new instance of PluginInitializer
func NewPluginInitializer(
	internalClient clientset.Interface,
	sharedInformers informers.SharedInformerFactory,
) admission.PluginInitializer {
	return pluginInitializer{
		internalClient: internalClient,
		informers:      sharedInformers,
	}
}

// Initialize checks the initialization interfaces implemented by each plugin
// and provide the appropriate initialization data
func (i pluginInitializer) Initialize(plugin admission.Interface) {
	if wants, ok := plugin.(WantsInternalMySQLClientSet); ok {
		wants.SetInternalMySQLClientSet(i.internalClient)
	}

	if wants, ok := plugin.(WantsInternalMySQLInformerFactory); ok {
		wants.SetInternalMySQLInformerFactory(i.informers)
	}
}
