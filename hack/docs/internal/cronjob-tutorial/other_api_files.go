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

const groupVersionIntro = `
// +kubebuilder:docs-gen:collapse=Apache License

/*
First, we have some *package-level* markers that denote that there are
Kubernetes objects in this package, and that this package represents the group
` + "`" + `batch.tutorial.kubebuilder.io` + "`" + `. The` + " `" + `object` + "`" + ` generator makes use of the
former, while the latter is used by the CRD generator to generate the right
metadata for the CRDs it creates from this package.
*/
`

const groupVersionSchema = `
/*
Then, we have the commonly useful variables that help us set up our Scheme.
Since we need to use all the types in this package in our controller, it's
helpful (and the convention) to have a convenient method to add all the types to
some other` + " `" + `Scheme` + "`" + `. SchemeBuilder makes this easy for us.
*/`
