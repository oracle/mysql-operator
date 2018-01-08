// Package controllers implements common interface for controllers.
package controllers

// Controller provides an interface for controller executors.
type Controller interface {
	// Run executes the controller blocking until it recieves on the
	// stopChan.
	Run(stopChan <-chan struct{})
}
