package app

import (
	genericapiserver "k8s.io/apiserver/pkg/server"
)

// Run runs the apiserver until a signal is recieved on the stop channel.
func Run(o *MySQLAPIServerOptions, stopCh <-chan struct{}) error {
	if err := o.Validate(); err != nil {
		return err
	}

	config, err := o.Config()
	if err != nil {
		return err
	}

	server, err := config.Complete().New()
	if err != nil {
		return err
	}

	server.GenericAPIServer.AddPostStartHook("start-mysql-internal-informers", func(context genericapiserver.PostStartHookContext) error {
		config.ExtraConfig.SharedInformerFactory.Start(context.StopCh)
		return nil
	})

	return server.GenericAPIServer.PrepareRun().Run(stopCh)
}
