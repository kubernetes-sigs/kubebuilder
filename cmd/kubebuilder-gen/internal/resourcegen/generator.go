/*
Copyright 2017 The Kubernetes Authors.

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

package resourcegen

import (
	"github.com/kubernetes-sigs/kubebuilder/cmd/internal/codegen"
	"k8s.io/gengo/generator"
)

type Generator struct{}

//
//// Returns a Generator for a versioned resource package e.g. pkg/apis/<group>/<version>
//func (g *Generator) GenerateInstall(apigroup *codegen.APIGroup, filename string) generator.Generator {
//	return &installGenerator{
//		generator.DefaultGen{OptionalName: filename},
//		apigroup,
//	}
//}

// Returns a Generator for a versioned resource package e.g. pkg/apis/<group>/<version>
func (g *Generator) GenerateVersionedResource(apiversion *codegen.APIVersion, apigroup *codegen.APIGroup, filename string) generator.Generator {
	return &versionedGenerator{
		generator.DefaultGen{OptionalName: filename},
		apiversion,
		apigroup,
	}
}

//// GenerateUnversionedResource returns a Generator for an unversioned resource package e.g. pkg/apis/<group>
//func (g *Generator) GenerateUnversionedResource(apigroup *codegen.APIGroup, filename string) generator.Generator {
//	return &unversionedGenerator{
//		generator.DefaultGen{OptionalName: filename},
//		apigroup,
//	}
//}

//// GenerateAPIs returns a Generator for the apis package e.g. pkg/apis
//func (g *Generator) GenerateAPIs(apis *codegen.APIs, filename string) generator.Generator {
//	return &apiGenerator{
//		generator.DefaultGen{OptionalName: filename},
//		apis,
//	}
//}
