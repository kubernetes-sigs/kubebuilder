package addon

import (
	"fmt"
	"path/filepath"
	"strings"

	"sigs.k8s.io/kubebuilder/pkg/model"
	"sigs.k8s.io/kubebuilder/pkg/model/file"
)

// ReplaceTypes replaces the API types with a modified version
func ReplaceTypes(u *model.Universe) error {
	funcs := DefaultTemplateFunctions()
	funcs["JSONTag"] = JSONTag

	contents, err := RunTemplate("types", typesTemplate, u, funcs)
	if err != nil {
		return err
	}

	var path string
	if u.Config.MultiGroup {
		path = filepath.Join("apis", u.Resource.Version, strings.ToLower(u.Resource.Kind)+"_types.go")
	} else {
		path = filepath.Join("api", u.Resource.Version, strings.ToLower(u.Resource.Kind)+"_types.go")
	}

	m := &file.File{
		Path:           path,
		Contents:       contents,
		IfExistsAction: file.Error,
	}

	ReplaceFileIfExists(u, m)

	return nil
}

// JSONTag is a helper to build the json tag for a struct
// It works around escaping problems for the json tag syntax
func JSONTag(tag string) string {
	return fmt.Sprintf("`json:\"%s\"`", tag)
}

// Resource.Resource

const typesTemplate = `{{ .Boilerplate }}

package {{ .Resource.Version }}

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	addonv1alpha1 "sigs.k8s.io/kubebuilder-declarative-pattern/pkg/patterns/addon/pkg/apis/v1alpha1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// {{.Resource.Kind}}Spec defines the desired state of {{.Resource.Kind}}
type {{.Resource.Kind}}Spec struct {
	addonv1alpha1.CommonSpec {{ JSONTag ",inline" }}
	addonv1alpha1.PatchSpec  {{ JSONTag ",inline" }}

	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// {{.Resource.Kind}}Status defines the observed state of {{.Resource.Kind}}
type {{.Resource.Kind}}Status struct {
	addonv1alpha1.CommonStatus {{ JSONTag ",inline" }}

	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
{{ if not .Resource.Namespaced }} // +kubebuilder:resource:scope=Cluster {{ end }}

// {{.Resource.Kind}} is the Schema for the {{ .Resource.Plural }} API
type {{.Resource.Kind}} struct {
	metav1.TypeMeta   ` + "`" + `json:",inline"` + "`" + `
	metav1.ObjectMeta ` + "`" + `json:"metadata,omitempty"` + "`" + `

	Spec   {{.Resource.Kind}}Spec   ` + "`" + `json:"spec,omitempty"` + "`" + `
	Status {{.Resource.Kind}}Status ` + "`" + `json:"status,omitempty"` + "`" + `
}

var _ addonv1alpha1.CommonObject = &{{.Resource.Kind}}{}

func (o *{{.Resource.Kind}}) ComponentName() string {
	return "{{ .Resource.Kind | lower }}"
}

func (o *{{.Resource.Kind}}) CommonSpec() addonv1alpha1.CommonSpec {
	return o.Spec.CommonSpec
}

func (o *{{.Resource.Kind}}) PatchSpec() addonv1alpha1.PatchSpec {
	return o.Spec.PatchSpec
}

func (o *{{.Resource.Kind}}) GetCommonStatus() addonv1alpha1.CommonStatus {
	return o.Status.CommonStatus
}

func (o *{{.Resource.Kind}}) SetCommonStatus(s addonv1alpha1.CommonStatus) {
	o.Status.CommonStatus = s
}

// +kubebuilder:object:root=true
{{ if not .Resource.Namespaced }} // +kubebuilder:resource:scope=Cluster {{ end }}

// {{.Resource.Kind}}List contains a list of {{.Resource.Kind}}
type {{.Resource.Kind}}List struct {
	metav1.TypeMeta ` + "`" + `json:",inline"` + "`" + `
	metav1.ListMeta ` + "`" + `json:"metadata,omitempty"` + "`" + `
	Items           []{{ .Resource.Kind }} ` + "`" + `json:"items"` + "`" + `
}

func init() {
	SchemeBuilder.Register(&{{.Resource.Kind}}{}, &{{.Resource.Kind}}List{})
}
`
