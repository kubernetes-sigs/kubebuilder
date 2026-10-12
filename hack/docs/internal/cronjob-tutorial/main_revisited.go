/*
Copyright 2023 The Kubernetes Authors.

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

package cronjob

const mainBatch = `
// +kubebuilder:docs-gen:collapse=Imports

/*
The first difference to notice is that kubebuilder has added the new API
group's package (` + "`" + `batchv1` + "`" + `) to our scheme.  This means that we can use those
objects in our controller.

If we would be using any other CRD we would have to add their scheme the same way.
Builtin types such as Job have their scheme added by` + " `" + `clientgoscheme` + "`" + `.
*/`

const mainEnableWebhook = `

	/*
		Register webhooks for our type with the manager, as explained in the next section.
		To run the controller locally without webhook certificates, disable the webhook
		server with` + " `" + `make run RUN_ARGS=--webhook-port=-1` + "`" + `.
	*/`
