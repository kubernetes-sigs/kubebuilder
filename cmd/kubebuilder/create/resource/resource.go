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
	"fmt"
	"path/filepath"
	"strings"

	createutil "github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/create/util"
	"github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/util"
)

type resourceTemplateArgs struct {
	BoilerPlate       string
	Domain            string
	Group             string
	Version           string
	Kind              string
	Resource          string
	Repo              string
	PluralizedKind    string
	NonNamespacedKind bool
}

func doResource(dir string, args resourceTemplateArgs) bool {
	typesFileName := fmt.Sprintf("%s_types.go", strings.ToLower(createutil.KindName))
	path := filepath.Join(dir, "pkg", "apis", createutil.GroupName, createutil.VersionName, typesFileName)
	fmt.Printf("\t%s\n", filepath.Join(
		"pkg", "apis", createutil.GroupName, createutil.VersionName, typesFileName))
	return util.WriteIfNotFound(path, "resource-template", resourceTemplate, args)
}

var resourceTemplate = `
{{.BoilerPlate}}

package {{.Version}}

import (
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!
// Created by "kubebuilder create resource" for you to implement the {{.Kind}} resource schema definition
// as a go struct.
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// {{.Kind}}Spec defines the desired state of {{.Kind}}
type {{.Kind}}Spec struct {
    // INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "kubebuilder generate" to regenerate code after modifying this file
}

// {{.Kind}}Status defines the observed state of {{.Kind}}
type {{.Kind}}Status struct {
    // INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "kubebuilder generate" to regenerate code after modifying this file
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
{{- if .NonNamespacedKind }}
// +genclient:nonNamespaced
{{- end }}

// {{.Kind}}
// +k8s:openapi-gen=true
// +kubebuilder:resource:path={{.Resource}}
type {{.Kind}} struct {
    metav1.TypeMeta   ` + "`" + `json:",inline"` + "`" + `
    metav1.ObjectMeta ` + "`" + `json:"metadata,omitempty"` + "`" + `

    Spec   {{.Kind}}Spec   ` + "`" + `json:"spec,omitempty"` + "`" + `
    Status {{.Kind}}Status ` + "`" + `json:"status,omitempty"` + "`" + `
}
`
