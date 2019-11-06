package webhook

import (
	"fmt"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/kubebuilder/pkg/model"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/resource"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/scaffoldtest"
)

var _ = Describe("Webhook", func() {
	type webhookTestcase struct {
		resource.Resource
		Config
	}

	serverName := "default"
	domainName := "testproject.org"
	inputs := []*webhookTestcase{
		{
			Resource: resource.Resource{Group: "crew", Version: "v1", Kind: "FirstMate", Namespaced: true, CreateExampleReconcileBody: true},
			Config: Config{
				Type:       "mutating",
				Operations: []string{"create", "update"},
				Server:     serverName,
			},
		},
		{
			Resource: resource.Resource{Group: "crew", Version: "v1", Kind: "FirstMate", Namespaced: true, CreateExampleReconcileBody: true},
			Config: Config{
				Type:       "mutating",
				Operations: []string{"delete"},
				Server:     serverName,
			},
		},
		{
			Resource: resource.Resource{Group: "ship", Version: "v1beta1", Kind: "Frigate", Namespaced: true, CreateExampleReconcileBody: false},
			Config: Config{
				Type:       "validating",
				Operations: []string{"update"},
				Server:     serverName,
			},
		},
		{
			Resource: resource.Resource{Group: "creatures", Version: "v2alpha1", Kind: "Kraken", Namespaced: false, CreateExampleReconcileBody: false},
			Config: Config{
				Type:       "validating",
				Operations: []string{"create"},
				Server:     serverName,
			},
		},
		{
			Resource: resource.Resource{Group: "core", Version: "v1", Kind: "Namespace", Namespaced: false, CreateExampleReconcileBody: false},
			Config: Config{
				Type:       "mutating",
				Operations: []string{"update"},
				Server:     serverName,
			},
		},
	}

	for i := range inputs {
		in := inputs[i]
		Describe(fmt.Sprintf("scaffolding webhook %s", in.Kind), func() {
			files := []struct {
				instance input.File
				file     string
			}{
				{
					file: filepath.Join("pkg", "webhook", "add_default_server.go"),
					instance: &AddServer{
						Resource: &in.Resource,
						Config:   in.Config,
					},
				},
				{
					file: filepath.Join("pkg", "webhook", "default_server", "server.go"),
					instance: &Server{
						Resource: &in.Resource,
						Config:   in.Config,
					},
				},
				{
					file: filepath.Join("pkg", "webhook", "default_server",
						fmt.Sprintf("add_%s_%s.go", strings.ToLower(in.Type), strings.ToLower(in.Kind))),
					instance: &AddAdmissionWebhookBuilderHandler{
						Resource: &in.Resource,
						Config:   in.Config,
					},
				},
				{
					file: filepath.Join("pkg", "webhook", "default_server",
						fmt.Sprintf("add_%s_%s.go", strings.ToLower(in.Type), strings.ToLower(in.Kind))),
					instance: &AddAdmissionWebhookBuilderHandler{
						Resource: &in.Resource,
						Config:   in.Config,
					},
				},
				{
					file: filepath.Join("pkg", "webhook", "default_server",
						strings.ToLower(in.Kind), strings.ToLower(in.Type),
						"webhooks.go"),
					instance: &AdmissionWebhooks{
						Resource: &in.Resource,
						Config:   in.Config,
					},
				},
				{
					file: filepath.Join("pkg", "webhook", "default_server",
						strings.ToLower(in.Kind), strings.ToLower(in.Type),
						fmt.Sprintf("%s_webhook.go", strings.Join(in.Operations, "_"))),
					instance: &AdmissionWebhookBuilder{
						Resource:    &in.Resource,
						Config:      in.Config,
						GroupDomain: domainName,
					},
				},
				{
					file: filepath.Join("pkg", "webhook", "default_server",
						strings.ToLower(in.Kind), strings.ToLower(in.Type),
						fmt.Sprintf("%s_%s_handler.go", strings.ToLower(in.Kind), strings.Join(in.Operations, "_"))),
					instance: &AdmissionHandler{
						Resource:    &in.Resource,
						Config:      in.Config,
						GroupDomain: domainName,
					},
				},
			}

			for j := range files {
				f := files[j]
				Context(f.file, func() {
					It(fmt.Sprintf("should write a file matching the golden file %s", f.file), func() {
						s, result := scaffoldtest.NewTestScaffold(f.file, f.file)
						Expect(s.Execute(&model.Universe{}, scaffoldtest.Options(), f.instance)).To(Succeed())
						Expect(result.Actual.String()).To(Equal(result.Golden), result.Actual.String())
					})
				})
			}
		})
	}
})
