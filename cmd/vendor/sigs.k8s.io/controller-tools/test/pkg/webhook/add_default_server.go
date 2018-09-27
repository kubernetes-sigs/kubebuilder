/*
Copyright 2018 The Kubernetes authors.

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

package webhook

import (
	server "sigs.k8s.io/controller-tools/test/pkg/webhook/default_server"
)

func init() {
	// AddToManagerFuncs is a list of functions to create webhook servers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, server.Add)
}
