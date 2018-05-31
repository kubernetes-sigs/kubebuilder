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

/*
The webhook package provides utilities to build 1) a admission webhook server
2) an admission webhook installer to manage the certificate and the k8s admission webhook resources

An example of how to build the admission webhook server

	as := DefaultAdmissionServer(ValidatingType, []metav1.GroupVersionResource{{Version: "v1", Resource: "pods"}})
	err := as.GetDefault()
	if err != nil {
		// handle error
	}
	for _, hc := range as.Config.ServerConfig {
		as.HandleFunc(hc, someHandlerFn) // someHandlerFn is a handler function defined by the user.
	}
	err = as.ListenAndServeTLS()
	if err != nil {
		// handle error
	}

Use the admission webhook server to generate the k8s resources for deploying

	resources, err := as.Generate()
	if err != nil {
		// handle error
	}
	// write resources to stdout or a file.

*/
package internal
