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
Package certprovider provides interface and implementation to provision
certificates.

Create a certprovider instance. For example:

	cp := SelfSignedCertProvider{
		// your configuration
	}

Generate and consume the certificates.

The certificates are stored in a secret, you can consume it by mounting the secret in a pod:
https://kubernetes.io/docs/concepts/configuration/secret/#using-secrets-as-files-from-a-pod
You then can pass the file names to method ListenAndServeTLS.

	_, _, err := cp.GetServerCert()
	if err != nil {
		// handle error
	}
	// key and cert are mounted as keyFile and certFile respectively.
	svr := &http.Server{
		// configure your server
	}
	svr.ListenAndServeTLS(certFile, keyFile)
*/
package certprovider
