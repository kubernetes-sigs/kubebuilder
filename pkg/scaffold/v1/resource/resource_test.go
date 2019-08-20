package resource_test

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/scaffoldtest"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/v1/resource"
)

func TestScaffoldingAPI(t *testing.T) {
	grid := []struct {
		Name             string
		Instance         *resource.Resource
		ExpectedError    string
		ExpectedResource string
	}{
		{
			Name:     "succeed if the Resource is valid",
			Instance: &resource.Resource{Group: "crew", Version: "v1", Kind: "FirstMate"},
		},
		{
			Name:          "fail if the Group is not specified",
			Instance:      &resource.Resource{Version: "v1", Kind: "FirstMate"},
			ExpectedError: "group cannot be empty",
		},
		{
			Name:          "fail if the Group is not all lowercase",
			Instance:      &resource.Resource{Group: "Crew", Version: "v1", Kind: "FirstMate"},
			ExpectedError: "group must match ^[a-z]+$ (was Crew)",
		},
		{
			Name:          "fail if the Group contains non-alpha characters",
			Instance:      &resource.Resource{Group: "crew1", Version: "v1", Kind: "FirstMate"},
			ExpectedError: "group must match ^[a-z]+$ (was crew1)",
		},
		{
			Name:          "fail if the Version is not specified",
			Instance:      &resource.Resource{Group: "crew", Kind: "FirstMate"},
			ExpectedError: "version cannot be empty",
		},
		{
			Name:          "fail if the Kind is not specified",
			Instance:      &resource.Resource{Group: "crew", Version: "v1"},
			ExpectedError: "kind cannot be empty",
		},
		{
			Name:             "allow Cat as a Kind",
			Instance:         &resource.Resource{Group: "crew", Kind: "Cat", Version: "v1"},
			ExpectedResource: "cats",
		},

		{
			Name:             "keep the Resource if specified",
			Instance:         &resource.Resource{Group: "crew", Kind: "FirstMate", Version: "v1", Resource: "myresource"},
			ExpectedResource: "myresource",
		},

		{
			Name:     "fail if the Kind is not camel cased (FirstMate - base case)",
			Instance: &resource.Resource{Group: "crew", Kind: "FirstMate", Version: "v1"},
		},

		{
			// Can't detect this case :(
			Name:     "fail if the Kind is not camel cased (Firstmate - cannot be detected)",
			Instance: &resource.Resource{Group: "crew", Kind: "Firstmate", Version: "v1"},
		},

		{
			Name:          "fail if the Kind is not camel cased (firstMate)",
			Instance:      &resource.Resource{Group: "crew", Kind: "firstMate", Version: "v1"},
			ExpectedError: "kind must be camelcase (expected FirstMate was firstMate)",
		},

		{
			Name:          "fail if the Kind is not camel cased (firstmate)",
			Instance:      &resource.Resource{Group: "crew", Kind: "firstmate", Version: "v1"},
			ExpectedError: "kind must be camelcase (expected Firstmate was firstmate)",
		},

		{
			Name:             "default the Resource by pluralizing the Kind (FirstMate)",
			Instance:         &resource.Resource{Group: "crew", Kind: "FirstMate", Version: "v1"},
			ExpectedResource: "firstmates",
		},

		{
			Name:             "default the Resource by pluralizing the Kind (fish)",
			Instance:         &resource.Resource{Group: "crew", Kind: "Fish", Version: "v1"},
			ExpectedResource: "fish",
		},

		{
			Name:             "default the Resource by pluralizing the Kind (Helmswoman)",
			Instance:         &resource.Resource{Group: "crew", Kind: "Helmswoman", Version: "v1"},
			ExpectedResource: "helmswomen",
		},
	}

	for _, g := range grid {
		g := g
		t.Run(fmt.Sprintf("%s resource=%+v", g.Name, g.Instance), func(t *testing.T) {
			err := g.Instance.Validate()
			if g.ExpectedError == "" {
				if err != nil {
					t.Errorf("validate failed unexpectedly: err=%v", err)
				}
			} else {
				if err == nil {
					t.Errorf("validate succeeded, but was expected to fail with %q", g.ExpectedError)
				} else {
					if !strings.Contains(err.Error(), g.ExpectedError) {
						t.Errorf("validate expected to fail with %q, but failed with %q", g.ExpectedError, err.Error())
					}
				}
			}

			if g.ExpectedResource != "" {
				if g.Instance.Resource != g.ExpectedResource {
					t.Errorf("unexpected Resource; expected %q, was %q", g.ExpectedResource, g.Instance.Resource)
				}
			}
		})
	}
}

func TestValidateVersion(t *testing.T) {
	versions := []string{
		"1",
		"1beta1",
		"a1beta1",
		"v1beta",
		"v1beta1alpha1",
	}

	for _, version := range versions {
		version := version
		t.Run(fmt.Sprintf("version=%s", version), func(t *testing.T) {
			instance := &resource.Resource{Group: "crew", Version: version, Kind: "FirstMate"}
			err := instance.Validate()
			if err == nil {
				t.Errorf("validate succeeded, but was expected to fail with version error")
			} else {
				expectedError := `version must match ^v\d+(alpha\d+|beta\d+)?$ (was ` + version + `)`
				if !strings.Contains(err.Error(), expectedError) {
					t.Errorf("validate expected to fail with %q, but failed with %q", expectedError, err.Error())
				}
			}
		})
	}
}

func TestScaffoldResources(t *testing.T) {
	resources := []*resource.Resource{
		{Group: "crew", Version: "v1", Kind: "FirstMate", Namespaced: true, CreateExampleReconcileBody: true},
		{Group: "ship", Version: "v1beta1", Kind: "Frigate", Namespaced: true, CreateExampleReconcileBody: false},
		{Group: "creatures", Version: "v2alpha1", Kind: "Kraken", Namespaced: false, CreateExampleReconcileBody: false},
	}

	for i := range resources {
		r := resources[i]
		t.Run(fmt.Sprintf("scaffolding API %s", r.Kind), func(t *testing.T) {
			files := []struct {
				instance input.File
				file     string
			}{
				{
					file: filepath.Join("pkg", "apis",
						fmt.Sprintf("addtoscheme_%s_%s.go", r.Group, r.Version)),
					instance: &resource.AddToScheme{Resource: r},
				},
				{
					file:     filepath.Join("pkg", "apis", r.Group, r.Version, "doc.go"),
					instance: &resource.Doc{Resource: r},
				},
				{
					file:     filepath.Join("pkg", "apis", r.Group, "group.go"),
					instance: &resource.Group{Resource: r},
				},
				{
					file:     filepath.Join("pkg", "apis", r.Group, r.Version, "register.go"),
					instance: &resource.Register{Resource: r},
				},
				{
					file: filepath.Join("pkg", "apis", r.Group, r.Version,
						strings.ToLower(r.Kind)+"_types.go"),
					instance: &resource.Types{Resource: r},
				},
				{
					file: filepath.Join("pkg", "apis", r.Group, r.Version,
						strings.ToLower(r.Kind)+"_types_test.go"),
					instance: &resource.TypesTest{Resource: r},
				},
				{
					file:     filepath.Join("pkg", "apis", r.Group, r.Version, r.Version+"_suite_test.go"),
					instance: &resource.VersionSuiteTest{Resource: r},
				},
				{
					file: filepath.Join("config", "samples",
						fmt.Sprintf("%s_%s_%s.yaml", r.Group, r.Version, strings.ToLower(r.Kind))),
					instance: &resource.CRDSample{Resource: r},
				},
			}

			for j := range files {
				f := files[j]
				t.Run(fmt.Sprintf("file %s", f.file), func(t *testing.T) {
					s, result := scaffoldtest.NewGoTestScaffold(t, f.file, f.file)
					if err := s.Execute(scaffoldtest.Options(), f.instance); err != nil {
						t.Fatalf("error from Execute: %v", err)
					}
					result.CheckGoldenOutput(t, result.Actual.String())
				})
			}
		})
	}
}
