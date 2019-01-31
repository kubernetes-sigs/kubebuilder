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
Package builder provides methods to build admission webhooks.

The following are 2 examples for building mutating webhook and validating webhook.

	webhook1 := NewWebhookBuilder().
		Mutating().
		Path("/mutatepods").
		ForType(&corev1.Pod{}).
		Handlers(mutatingHandler11, mutatingHandler12).
		Build()

	webhook2 := NewWebhookBuilder().
		Validating().
		Path("/validatepods").
		ForType(&appsv1.Deployment{}).
		Handlers(validatingHandler21).
		Build()

Note: To build a webhook for a CRD, you need to ensure the manager uses the scheme that understands your CRD.
This is necessary, because if the scheme doesn't understand your CRD types, the decoder won't be able to decode
the CR object from the admission review request.

The following snippet shows how to register CRD types with manager's scheme.

	mgr, err := manager.New(cfg, manager.Options{})
	if err != nil {
		// handle error
	}
	// SchemeGroupVersion is group version used to register these objects
	SchemeGroupVersion = schema.GroupVersion{Group: "crew.k8s.io", Version: "v1"}
	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion}
	// Register your CRD types.
	SchemeBuilder.Register(&Kraken{}, &KrakenList{})
	// Register your CRD types with the manager's scheme.
	err = SchemeBuilder.AddToScheme(mgr.GetScheme())
	if err != nil {
		// handle error
	}
*/
package builder
