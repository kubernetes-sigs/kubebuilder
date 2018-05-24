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

package certprovider

import (
	"net/http"
)

func ExampleNewSelfSignedCertProvider() {
	cp := NewSelfSignedCertProvider()

	err := cp.GenerateCA()
	if err != nil {
		// handle error
	}
	err = cp.PersistCA("myNamespace", "mySecret", "ca-key.pem", "ca-cert.pem")
	if err != nil {
		// handle error
	}

	err = cp.GenerateServerCert("k8s.io", []string{"myDNSName"}, 365)
	if err != nil {
		// handle error
	}
	err = cp.PersistServerCert("myNamespace", "mySecret", "server-key.pem", "server-cert.pem")
	if err != nil {
		// handle error
	}

	svr := &http.Server{
		// server configuration
	}
	svr.ListenAndServeTLS("path/to/server-cert.pem", "path/to/server-key.pem")
}
