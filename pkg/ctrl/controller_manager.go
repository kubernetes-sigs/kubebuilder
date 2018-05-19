/*
Copyright 2018 The Kubernetes Authors.

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

package ctrl

// DefaultControllerManager is the default ControllerManager.
var DefaultControllerManager = &ControllerManager{}

// ControllerManager initializes and starts Controllers.  ControllerManager should be used if there are multiple
// Controllers to share caches, stop channels, and other shared dependencies across Controllers.
type ControllerManager struct {
	// Stop may be closed to shutdown all Controllers.  Defaults to a new channel.
	Stop <-chan struct{}

	controllers []*Controller
}

// Register registers a Controller with the ControllerManager.
// The ControllerManager Stop channel will be set on each Controller when it is registered.
func (cm ControllerManager) Register(c *Controller) {
	cm.controllers = append(cm.controllers, c)
}

// Start starts all registered Controllers and blocks until the Stop channel is closed.
// Returns an error if there is an error starting any Controller.
func (cm ControllerManager) Start() error {
	// Default the Stop channel
	if cm.Stop == nil {
		cm.Stop = make(<-chan struct{})
	}

	// Start each controller
	for _, c := range cm.controllers {
		// Default the controller stop channel
		if c.Stop == nil {
			c.Stop = cm.Stop
		}
		// TODO: write starting Controllers so we don't block
	}
	<-cm.Stop
	return nil
}

// Register registers a Controller with the DefaultControllerManager.
func RegisterController(c *Controller) { DefaultControllerManager.Register(c) }

// Start starts all Controllers registered with the DefaultControllerManager.
func Start() error { return DefaultControllerManager.Start() }
