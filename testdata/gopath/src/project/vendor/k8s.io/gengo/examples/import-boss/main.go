/*
Copyright 2016 The Kubernetes Authors.

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

// import-boss enforces import restrictions in a given repository.
//
// When a package is verified, import-boss looks for files called
// ".import-restrictions" in all directories between the package and
// the $GOPATH/src.
//
// All imports of the package are checked against each "rule" in the
// found restriction files, climbing up the directory tree until the
// import matches one of the rules.
//
// If the import does not match any of the rules, it is accepted.
//
// In analogy, all incoming imports of the package are checked against each
// "inverse rule" in the found restriction files, climbing up the directory
// tree until the import matches one of the rules.
//
// If the incoming import does not match any of the inverse rules, it is accepted.
//
// A rule consists of three parts:
// * A SelectorRegexp, to select the import paths that the rule applies to.
// * A list of AllowedPrefixes
// * A list of ForbiddenPrefixes
// An import passes a rule of a matching selector if it matches at least one
// allowed prefix, but no forbidden prefix.
//
// An example file looks like this:
//
// {
//   "Rules": [
//     {
//       "SelectorRegexp": "k8s[.]io",
//       "AllowedPrefixes": [
//         "k8s.io/gengo/examples",
//         "k8s.io/kubernetes/third_party"
//       ],
//       "ForbiddenPrefixes": [
//         "k8s.io/kubernetes/pkg/third_party/deprecated"
//       ]
//     },
//     {
//       "SelectorRegexp": "^unsafe$",
//       "AllowedPrefixes": [
//       ],
//       "ForbiddenPrefixes": [
//         ""
//       ]
//     }
//   ],
//   "InverseRules": [{
//       "SelectorRegexp": "k8s[.]io",
//       "AllowedPrefixes": [
//         "k8s.io/same-repo",
//         "k8s.io/kubernetes/pkg/legacy"
//       ],
//       "ForbiddenPrefixes": [
//         "k8s.io/kubernetes/pkg/legacy/subpkg"
//       ]
//     },
//     {
//       "SelectorRegexp": "k8s[.]io",
//       "Transitive": true,
//       "AllowedPrefixes": [
//         "k8s.io/
//       ],
//       "ForbiddenPrefixes": [
//         "k8s.io/kubernetes/cmd/kubelet",
//         "k8s.io/kubernetes/cmd/kubectl"
//       ],
//   ]
// }
//
// Note the second (non-inverse) rule explicitly matches the unsafe package, and forbids it
// ("" is a prefix of everything).
//
// An import from another package passes an inverse rule with a matching selector if
// it matches at least one allowed prefix, but no forbidden prefix.
//
// Note that the second InverseRule is transitive, the first only applies to direct imports.
package main

import (
	"os"

	"k8s.io/gengo/args"
	"k8s.io/gengo/examples/import-boss/generators"

	"k8s.io/klog"
)

func main() {
	klog.InitFlags(nil)
	arguments := args.Default()
	if err := arguments.Execute(
		generators.NameSystems(),
		generators.DefaultNameSystem(),
		generators.Packages,
	); err != nil {
		klog.Errorf("Error: %v", err)
		os.Exit(1)
	}
	klog.V(2).Info("Completed successfully.")
}
