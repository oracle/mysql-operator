package app

import (
	"fmt"
	"io"
	"net"

	"github.com/spf13/pflag"

	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	genericapiserver "k8s.io/apiserver/pkg/server"
	genericoptions "k8s.io/apiserver/pkg/server/options"

	"github.com/oracle/mysql-operator/pkg/api"
	"github.com/oracle/mysql-operator/pkg/apis/mysql/v1"
	"github.com/oracle/mysql-operator/pkg/apiserver"
	"github.com/oracle/mysql-operator/pkg/apiserver/admission"
	clientset "github.com/oracle/mysql-operator/pkg/generated/clientset/internalversion"
	informers "github.com/oracle/mysql-operator/pkg/generated/informers/internalversion"
	"github.com/oracle/mysql-operator/pkg/version"
)

const defaultEtcdPathPrefix = "/registry/mysql.oracle.com"

type MySQLAPIServerOptions struct {
	RecommendedOptions *genericoptions.RecommendedOptions
	Admission          *genericoptions.AdmissionOptions

	// StandaloneMode if true asserts that we will not depend on a kube-apiserver
	StandaloneMode bool

	// whether or not to serve the OpenAPI spec (at /swagger.json)
	ServeOpenAPISpec bool

	StdOut io.Writer
	StdErr io.Writer
}

func (o *MySQLAPIServerOptions) AddFlags(flags *pflag.FlagSet) {
	flags.BoolVar(
		&o.StandaloneMode,
		"standalone-mode",
		o.StandaloneMode,
		"Run the apiserver in standalone mode which has limited functionality and no dependencies on kube-apiserver.",
	)

	flags.BoolVar(
		&o.ServeOpenAPISpec,
		"serve-openapi-spec",
		false,
		"Whether this API server should serve the OpenAPI spec (problematic with older versions of kubectl)",
	)

	o.RecommendedOptions.AddFlags(flags)
	o.Admission.AddFlags(flags)
}

func NewMySQLAPIServerOptions(out, errOut io.Writer) *MySQLAPIServerOptions {
	o := &MySQLAPIServerOptions{
		RecommendedOptions: genericoptions.NewRecommendedOptions(defaultEtcdPathPrefix, api.Codecs.LegacyCodec(v1.SchemeGroupVersion)),
		Admission:          genericoptions.NewAdmissionOptions(),

		StdOut: out,
		StdErr: errOut,
	}

	o.RecommendedOptions.SecureServing.ServerCert.CertDirectory = "/var/run/mysql-operator"

	return o
}

func (o MySQLAPIServerOptions) Validate() error {
	errors := []error{}
	errors = append(errors, o.RecommendedOptions.Validate()...)
	errors = append(errors, o.Admission.Validate()...)
	return utilerrors.NewAggregate(errors)
}

func (o MySQLAPIServerOptions) Config() (*apiserver.Config, error) {
	// register admission plugins

	// TODO: external address?
	if err := o.RecommendedOptions.SecureServing.MaybeDefaultWithSelfSignedCerts("localhost", nil, []net.IP{net.ParseIP("127.0.0.1")}); err != nil {
		return nil, fmt.Errorf("error creating self-signed certificates: %v", err)
	}

	serverConfig := genericapiserver.NewRecommendedConfig(api.Codecs)
	genericConfig := &serverConfig.Config
	if err := applyOptions(
		genericConfig,
		o.RecommendedOptions.Etcd.ApplyTo,
		o.RecommendedOptions.Features.ApplyTo,
		o.RecommendedOptions.Audit.ApplyTo,
		o.RecommendedOptions.SecureServing.ApplyTo,
	); err != nil {
		return nil, err
	}

	client, err := clientset.NewForConfig(serverConfig.LoopbackClientConfig)
	if err != nil {
		return nil, err
	}
	informerFactory := informers.NewSharedInformerFactory(client, serverConfig.LoopbackClientConfig.Timeout)

	if !o.StandaloneMode {
		if err := applyOptions(
			genericConfig,
			o.RecommendedOptions.Authentication.ApplyTo,
			o.RecommendedOptions.Authorization.ApplyTo,
		); err != nil {
			return nil, err
		}

		if err := o.RecommendedOptions.CoreAPI.ApplyTo(serverConfig); err != nil {
			return nil, err
		}

		admissionInitializer := admission.NewPluginInitializer(
			client,
			informerFactory,
		)

		if err := o.Admission.ApplyTo(genericConfig, serverConfig.SharedInformerFactory, serverConfig.ClientConfig, api.Scheme, admissionInitializer); err != nil {
			return nil, err
		}
	}

	/*
	 *if o.ServeOpenAPISpec {
	 *    genericConfig.OpenAPIConfig = genericapiserver.DefaultOpenAPIConfig(
	 *        openapi.GetOpenAPIDefinitions, api.Scheme)
	 *    if genericConfig.OpenAPIConfig.Info == nil {
	 *        genericConfig.OpenAPIConfig.Info = &spec.Info{}
	 *    }
	 *    if genericConfig.OpenAPIConfig.Info.Version == "" {
	 *        if genericConfig.Version != nil {
	 *            genericConfig.OpenAPIConfig.Info.Version = strings.Split(genericConfig.Version.String(), "-")[0]
	 *        } else {
	 *            genericConfig.OpenAPIConfig.Info.Version = "unversioned"
	 *        }
	 *    }
	 *} else {
	 *    glog.Warning("OpenAPI spec will not be served")
	 *}
	 */

	genericConfig.SwaggerConfig = genericapiserver.DefaultSwaggerConfig()
	// TODO: investigate if we need metrics unique to service catalog, but take defaults for now
	// see https://github.com/kubernetes-incubator/service-catalog/issues/677
	genericConfig.EnableMetrics = true

	// if err := o.RecommendedOptions.ApplyTo(serverConfig); err != nil {
	// 	return nil, err
	// }

	// TODO: add support to default these values in build
	mysqlVersion := version.Get()
	serverConfig.Version = &mysqlVersion

	return &apiserver.Config{
		GenericConfig: serverConfig,
		ExtraConfig: apiserver.ExtraConfig{
			SharedInformerFactory: informerFactory,
		},
	}, nil
}

func applyOptions(config *genericapiserver.Config, applyTo ...func(*genericapiserver.Config) error) error {
	for _, fn := range applyTo {
		if err := fn(config); err != nil {
			return err
		}
	}
	return nil
}
