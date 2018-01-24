package apiserver

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"

	api "github.com/oracle/mysql-operator/pkg/api"
	mysql "github.com/oracle/mysql-operator/pkg/apis/mysql"
	v1 "github.com/oracle/mysql-operator/pkg/apis/mysql/v1"
	informers "github.com/oracle/mysql-operator/pkg/generated/informers/internalversion"
	mysqlregistry "github.com/oracle/mysql-operator/pkg/registry"
	clusterstorage "github.com/oracle/mysql-operator/pkg/registry/mysql/cluster"
)

type ExtraConfig struct {
	SharedInformerFactory informers.SharedInformerFactory
}

type Config struct {
	GenericConfig *genericapiserver.RecommendedConfig
	ExtraConfig   ExtraConfig
}

// MySQLServer contains state for a Kubernetes cluster master/api server.
type MySQLServer struct {
	GenericAPIServer *genericapiserver.GenericAPIServer
}

type completedConfig struct {
	GenericConfig genericapiserver.CompletedConfig
	ExtraConfig   *ExtraConfig
}

type CompletedConfig struct {
	// Embed a private pointer that cannot be instantiated outside of this package.
	*completedConfig
}

// Complete fills in any fields not set that are required to have valid data. It's mutating the receiver.
func (cfg *Config) Complete() CompletedConfig {
	c := completedConfig{
		cfg.GenericConfig.Complete(),
		&cfg.ExtraConfig,
	}

	c.GenericConfig.Version = &version.Info{
		Major: "1",
		Minor: "0",
	}

	return CompletedConfig{&c}
}

// New returns a new instance of MySQLServer from the given config.
func (c completedConfig) New() (*MySQLServer, error) {
	genericServer, err := c.GenericConfig.New("mysql-apiserver", genericapiserver.EmptyDelegate)
	if err != nil {
		return nil, err
	}

	s := &MySQLServer{
		GenericAPIServer: genericServer,
	}

	apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(mysql.GroupName, api.Registry, api.Scheme, metav1.ParameterCodec, api.Codecs)
	apiGroupInfo.GroupMeta.GroupVersion = v1.SchemeGroupVersion
	v1storage := map[string]rest.Storage{}

	clusterStore, clusterStatusStore := mysqlregistry.RESTInPeace(clusterstorage.NewREST(api.Scheme, c.GenericConfig.RESTOptionsGetter))

	v1storage["mysqlclusters"] = clusterStore
	v1storage["mysqlclusters/status"] = clusterStatusStore

	apiGroupInfo.VersionedResourcesStorageMap["v1"] = v1storage

	if err := s.GenericAPIServer.InstallAPIGroup(&apiGroupInfo); err != nil {
		return nil, err
	}

	return s, nil
}
