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

package cmd

import (
	"log/slog"
	"os"

	"github.com/spf13/afero"

	"sigs.k8s.io/kubebuilder/v4/internal/version"
	"sigs.k8s.io/kubebuilder/v4/pkg/cli"
	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v4/pkg/logging"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
	kustomizecommonv2 "sigs.k8s.io/kubebuilder/v4/pkg/plugins/common/kustomize/v2"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang"
	deployimagev1alpha1 "sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/deploy-image/v1alpha1"
	serversideapplyv1alpha1 "sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/server-side-apply/v1alpha1"
	golangv4 "sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/v4"
	autoupdatev1alpha "sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/autoupdate/v1alpha"
	grafanav1alpha "sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/grafana/v1alpha"
	helmv1alpha "sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v1alpha"
	helmv2alpha "sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v2alpha"
)

// Run bootstraps & runs the CLI
func Run() {
	// Initialize custom logging handler FIRST - applies to ALL CLI operations
	opts := logging.HandlerOptions{
		SlogOpts: slog.HandlerOptions{
			Level: slog.LevelInfo,
		},
	}
	handler := logging.NewHandler(os.Stdout, opts)
	logger := slog.New(handler)
	slog.SetDefault(logger)
	// Bundle plugin which built the golang projects scaffold with base.go/v4 and kustomize/v2 plugins
	gov4Bundle, _ := plugin.NewBundleWithOptions(plugin.WithName(golang.DefaultNameQualifier),
		plugin.WithVersion(plugin.Version{Number: 4}),
		plugin.WithPlugins(kustomizecommonv2.Plugin{}, golangv4.Plugin{}),
		plugin.WithDescription("Default scaffold (go/v4 + kustomize/v2)"),
	)

	fs := machinery.Filesystem{
		FS: afero.NewOsFs(),
	}
	externalPlugins, err := cli.DiscoverExternalPlugins(fs.FS)
	if err != nil {
		slog.Error("error discovering external plugins", "error", err)
	}

	v := version.New()
	c, err := cli.New(
		cli.WithCommandName("kubebuilder"),
		cli.WithVersion(v.PrintVersion()),
		cli.WithCliVersion(v.GetKubeBuilderVersion()),
		cli.WithPlugins(
			golangv4.Plugin{},
			gov4Bundle,
			&kustomizecommonv2.Plugin{},
			&deployimagev1alpha1.Plugin{},
			&serversideapplyv1alpha1.Plugin{},
			&grafanav1alpha.Plugin{},
			&helmv1alpha.Plugin{},
			&helmv2alpha.Plugin{},
			&autoupdatev1alpha.Plugin{},
		),
		cli.WithPlugins(externalPlugins...),
		cli.WithDefaultPlugins(cfgv3.Version, gov4Bundle),
		cli.WithDefaultProjectVersion(cfgv3.Version),
		cli.WithCompletion(),
	)
	if err != nil {
		slog.Error("failed to create CLI", "error", err)
		os.Exit(1)
	}
	if err := c.Run(); err != nil {
		slog.Error("CLI run failed", "error", err)
		os.Exit(1)
	}
}
