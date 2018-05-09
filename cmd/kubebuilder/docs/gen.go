package docs

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kubernetes-sigs/kubebuilder/cmd/internal/codegen/parse"
	"github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/util"
	"k8s.io/gengo/args"
)

// CodeGenerator generates code for Kubernetes resources and controllers
type CodeGenerator struct{}

// Execute parses packages and executes the code generators against the resource and controller packages
func (g CodeGenerator) Execute(dir string) error {
	arguments := args.Default()
	b, err := arguments.NewBuilder()
	if err != nil {
		return fmt.Errorf("Failed making a parser: %v", err)
	}
	for _, d := range []string{"./pkg/apis", "./pkg/controller", "./pkg/inject"} {
		if err := b.AddDirRecursive(d); err != nil {
			return fmt.Errorf("Failed making a parser: %v", err)
		}
	}
	c, err := parse.NewContext(b)
	if err != nil {
		return fmt.Errorf("Failed making a context: %v", err)
	}

	p := parse.NewAPIs(c, arguments)
	path := filepath.Join(dir, outputDir, "config.yaml")

	args := ConfigArgs{}
	groups := []string{}
	for group, _ := range p.ByGroupKindVersion {
		groups = append(groups, group)
	}
	sort.Strings(groups)
	for _, group := range groups {
		kindversion := p.ByGroupKindVersion[group]
		args.Groups = append(args.Groups, strings.Title(group))
		c := Category{
			Name:    strings.Title(group),
			Include: strings.ToLower(group),
		}
		m := map[string]Resource{}
		s := []string{}
		for kind, version := range kindversion {
			r := Resource{
				Group: group,
				Kind:  kind,
			}
			vs := []string{}
			for version := range version {
				vs = append(vs, version)
			}
			sort.Strings(vs)
			r.Version = vs[0]
			m[r.Kind] = r
			s = append(s, r.Kind)
		}
		// Sort the resources by name
		sort.Strings(s)
		for _, k := range s {
			c.Resources = append(c.Resources, m[k])
		}

		args.Categories = append(args.Categories, c)
	}

	os.Remove(path)
	util.Write(path, "docs-config-template", docsConfigTemplate, args)
	return nil
}

type ConfigArgs struct {
	Groups     []string
	Categories []Category
}

type Category struct {
	Name      string
	Include   string
	Resources []Resource
}

type Resource struct {
	Kind, Version, Group string
}

var docsConfigTemplate = `example_location: "examples"
api_groups: {{ range $group := .Groups }}
  - "{{ $group }}"
{{ end -}}
resource_categories: {{ range $category := .Categories }}
- name: "{{ $category.Name }}"
  include: "{{ $category.Include}}"
  resources: {{ range $resource := $category.Resources }}
  - name: "{{ $resource.Kind }}"
    version: "{{ $resource.Version }}"
    group: "{{ $resource.Group }}"{{ end }}
{{ end }}`
