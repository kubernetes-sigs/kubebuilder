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

package resource

import (
	"log"
	"os"
	"path/filepath"

	createutil "github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/create/util"
	"github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/util"
)

func createVersion(boilerplate string) {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatalf("%v\n", err)
		os.Exit(-1)
	}
	path := filepath.Join(dir, "pkg", "apis", createutil.GroupName, createutil.VersionName, "doc.go")
	util.WriteIfNotFound(path, "version-template", versionTemplate, versionTemplateArgs{
		boilerplate,
		util.Domain,
		createutil.GroupName,
		createutil.VersionName,
		util.Repo,
	})
}

type versionTemplateArgs struct {
	BoilerPlate string
	Domain      string
	Group       string
	Version     string
	Repo        string
}

var versionTemplate = `
{{.BoilerPlate}}

// Api versions allow the api contract for a resource to be changed while keeping
// backward compatibility by support multiple concurrent versions
// of the same resource

// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=package,register
// +k8s:conversion-gen={{.Repo}}/pkg/apis/{{.Group}}
// +k8s:defaulter-gen=TypeMeta
// +groupName={{.Group}}.{{.Domain}}
package {{.Version}} // import "{{.Repo}}/pkg/apis/{{.Group}}/{{.Version}}"
`
