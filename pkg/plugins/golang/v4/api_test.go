/*
Copyright 2026 The Kubernetes Authors.

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

package v4

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
	goPlugin "sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang"
)

var _ = Describe("createAPISubcommand", func() {
	var (
		subCmd *createAPISubcommand
		cfg    config.Config
		res    *resource.Resource
		tmpDir string
		cwd    string
	)

	BeforeEach(func() {
		subCmd = &createAPISubcommand{}
		cfg = cfgv3.New()
		_ = cfg.SetRepository("github.com/example/test")

		subCmd.options = &goPlugin.Options{}
		subCmd.resourceFlag = &pflag.Flag{Changed: true}
		subCmd.controllerFlag = &pflag.Flag{Changed: true}

		res = &resource.Resource{
			GVK: resource.GVK{
				Group:   "crew",
				Domain:  "test.io",
				Version: "v1",
				Kind:    "Captain",
			},
			Plural:   "captains",
			API:      &resource.API{},
			Webhooks: &resource.Webhooks{},
		}

		var err error
		cwd, err = os.Getwd()
		Expect(err).NotTo(HaveOccurred())

		tmpDir, err = os.MkdirTemp("", "kb-create-api-test-")
		Expect(err).NotTo(HaveOccurred())
		Expect(os.MkdirAll(filepath.Join(tmpDir, "cmd"), 0o755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(tmpDir, "cmd", "main.go"), []byte("package main"), 0o644)).To(Succeed())
		Expect(os.Chdir(tmpDir)).To(Succeed())

		Expect(subCmd.InjectConfig(cfg)).To(Succeed())
	})

	AfterEach(func() {
		Expect(os.Chdir(cwd)).To(Succeed())
		Expect(os.RemoveAll(tmpDir)).To(Succeed())
	})

	preScaffold := func() error {
		return subCmd.PreScaffold(machinery.Filesystem{})
	}

	It("should reject external API options when creating API in project", func() {
		subCmd.options.DoAPI = true
		subCmd.options.ExternalAPIPath = "github.com/external/api"

		Expect(subCmd.InjectResource(res)).To(Succeed())
		err := preScaffold()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("cannot use '--external-api-path'"))
	})

	It("should require external-api-path when using external-api-module", func() {
		subCmd.options.DoAPI = false
		subCmd.options.ExternalAPIModule = "github.com/external/api@v1.0.0"
		subCmd.options.ExternalAPIPath = ""

		Expect(subCmd.InjectResource(res)).To(Succeed())
		err := preScaffold()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("requires '--external-api-path'"))
	})

	It("should prevent duplicate API without force flag", func() {
		subCmd.options.DoAPI = true
		subCmd.options.DoController = true

		resWithAPI := *res
		resWithAPI.API = &resource.API{CRDVersion: "v1"}
		Expect(cfg.AddResource(resWithAPI)).To(Succeed())

		subCmd.force = false
		Expect(subCmd.InjectResource(res)).To(Succeed())
		err := preScaffold()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("API resource already exists"))
	})

	It("should allow duplicate API with force flag", func() {
		subCmd.options.DoAPI = true
		subCmd.options.DoController = true

		resWithAPI := *res
		resWithAPI.API = &resource.API{CRDVersion: "v1"}
		Expect(cfg.AddResource(resWithAPI)).To(Succeed())

		subCmd.force = true
		Expect(subCmd.InjectResource(res)).To(Succeed())
		Expect(preScaffold()).To(Succeed())
	})

	It("should prevent multiple groups in single-group project", func() {
		subCmd.options.DoAPI = true
		subCmd.options.DoController = true

		firstRes := resource.Resource{
			GVK: resource.GVK{
				Group:   "ship",
				Domain:  "test.io",
				Version: "v1",
				Kind:    "Frigate",
			},
			Plural: "frigates",
			API:    &resource.API{CRDVersion: "v1"},
		}
		Expect(cfg.AddResource(firstRes)).To(Succeed())

		res.Group = "crew"
		res.Plural = "captains"

		Expect(subCmd.InjectResource(res)).To(Succeed())
		err := preScaffold()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("multiple groups are not allowed"))
	})

	It("should allow multiple groups when multigroup is enabled", func() {
		subCmd.options.DoAPI = true
		subCmd.options.DoController = true

		Expect(cfg.SetMultiGroup()).To(Succeed())

		firstRes := resource.Resource{
			GVK: resource.GVK{
				Group:   "ship",
				Domain:  "test.io",
				Version: "v1",
				Kind:    "Frigate",
			},
			Plural: "frigates",
			API:    &resource.API{CRDVersion: "v1"},
		}
		Expect(cfg.AddResource(firstRes)).To(Succeed())

		res.Group = "crew"

		Expect(subCmd.InjectResource(res)).To(Succeed())
		Expect(preScaffold()).To(Succeed())
	})

	It("should reject duplicate controller name without force", func() {
		subCmd.options.DoAPI = false
		subCmd.options.DoController = true
		subCmd.options.Controller.Name = "captain-main"
		subCmd.force = true

		Expect(cfg.AddResource(resource.Resource{
			GVK: res.GVK,
			Controllers: []resource.Controller{
				{Name: "captain-main"},
			},
		})).To(Succeed())

		Expect(subCmd.InjectResource(res)).To(Succeed())
		Expect(preScaffold()).To(Succeed())
	})

	It("should allow additional controllers for external types", func() {
		subCmd.options.DoAPI = false
		subCmd.options.DoController = true
		subCmd.options.Controller.Name = "certificate-backup"
		subCmd.options.ExternalAPIPath = "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
		subCmd.options.ExternalAPIDomain = "io"

		res.Group = "cert-manager"
		res.Kind = "Certificate"
		res.Domain = "io"

		Expect(cfg.AddResource(resource.Resource{
			GVK: res.GVK,
			Controllers: []resource.Controller{
				{Name: "certificate-main"},
			},
			External: true,
			Path:     "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1",
		})).To(Succeed())

		Expect(subCmd.InjectResource(res)).To(Succeed())
		Expect(preScaffold()).To(Succeed())
	})
})
