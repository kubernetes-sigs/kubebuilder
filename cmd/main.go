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

package main

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"sigs.k8s.io/kubebuilder/v3/pkg/cli"
	cfgv2 "sigs.k8s.io/kubebuilder/v3/pkg/config/v2"
	cfgv3 "sigs.k8s.io/kubebuilder/v3/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"
	kustomizecommonv1 "sigs.k8s.io/kubebuilder/v3/pkg/plugins/common/kustomize/v1"
	kustomizecommonv2alpha "sigs.k8s.io/kubebuilder/v3/pkg/plugins/common/kustomize/v2"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang"

	//nolint:staticcheck
	declarativev1 "sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/declarative/v1"
	deployimagev1alpha1 "sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/deploy-image/v1alpha1"
	golangv2 "sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v2"
	golangv3 "sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v3"
	golangv4 "sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v4"
	grafanav1alpha1 "sigs.k8s.io/kubebuilder/v3/pkg/plugins/optional/grafana/v1alpha"
)

func init() {
	// Disable timestamps on the default TextFormatter
	logrus.SetFormatter(&logrus.TextFormatter{DisableTimestamp: true})
}

func main() {

	const deprecateMessageGoV3Bundle = "This version is deprecated." +
		"The `go/v3` cannot scaffold projects using kustomize versions v4x+" +
		" and cannot fully support Kubernetes 1.25+." +
		"It is recommended to upgrade your project to the latest versions available (go/v4)." +
		"Please, check the migration guide to learn how to upgrade your project"

	// Bundle plugin which built the golang projects scaffold by Kubebuilder go/v3
	gov3Bundle, _ := plugin.NewBundleWithOptions(plugin.WithName(golang.DefaultNameQualifier),
		plugin.WithVersion(plugin.Version{Number: 3}),
		plugin.WithDeprecationMessage(deprecateMessageGoV3Bundle),
		plugin.WithPlugins(kustomizecommonv1.Plugin{}, golangv3.Plugin{}),
	)

	// Bundle plugin which built the golang projects scaffold by Kubebuilder go/v4 with kustomize alpha-v2
	gov4Bundle, _ := plugin.NewBundleWithOptions(plugin.WithName(golang.DefaultNameQualifier),
		plugin.WithVersion(plugin.Version{Number: 4}),
		plugin.WithPlugins(kustomizecommonv2alpha.Plugin{}, golangv4.Plugin{}),
	)

	fs := machinery.Filesystem{
		FS: afero.NewOsFs(),
	}
	externalPlugins, err := cli.DiscoverExternalPlugins(fs.FS)
	if err != nil {
		logrus.Error(err)
	}

	c, err := cli.New(
		cli.WithCommandName("kubebuilder"),
		cli.WithVersion(versionString()),
		cli.WithPlugins(
			golangv2.Plugin{},
			golangv3.Plugin{},
			golangv4.Plugin{},
			gov3Bundle,
			gov4Bundle,
			&kustomizecommonv1.Plugin{},
			&kustomizecommonv2alpha.Plugin{},
			&declarativev1.Plugin{},
			&deployimagev1alpha1.Plugin{},
			&grafanav1alpha1.Plugin{},
		),
		cli.WithPlugins(externalPlugins...),
		cli.WithDefaultPlugins(cfgv2.Version, golangv2.Plugin{}),
		cli.WithDefaultPlugins(cfgv3.Version, gov4Bundle),
		cli.WithDefaultProjectVersion(cfgv3.Version),
		cli.WithCompletion(),
	)
	if err != nil {
		logrus.Fatal(err)
	}
	if err := c.Run(); err != nil {
		logrus.Fatal(err)
	}
}
