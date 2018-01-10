/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package signals

import (
	"context"
	"os"
	"os/signal"
)

var onlyOneSignalHandler = make(chan struct{})

// SetupSignalHandler sets up a signal handler that calls the given CancelFunc
// on SIGTERM/SIGINT. If a second signal is caught, the program is terminated
// immediately with exit code 1.
func SetupSignalHandler(cancelFunc context.CancelFunc) {
	close(onlyOneSignalHandler) // panics when called twice

	c := make(chan os.Signal, 2)
	signal.Notify(c, shutdownSignals...)
	go func() {
		<-c
		cancelFunc()
		<-c
		os.Exit(1) // second signal. Exit directly.
	}()
}
