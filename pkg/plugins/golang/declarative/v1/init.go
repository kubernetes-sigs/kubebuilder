/*
Copyright 2021 The Kubernetes Authors.

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

package v1

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin/util"
)

var _ plugin.InitSubcommand = &initSubcommand{}

type initSubcommand struct {
	config config.Config
}

func (p *initSubcommand) InjectConfig(c config.Config) error {
	p.config = c

	return nil
}

func (p *initSubcommand) Scaffold(_ machinery.Filesystem) error {
	//nolint:staticcheck
	err := updateDockerfile(plugin.IsLegacyLayout(p.config))
	if err != nil {
		return err
	}
	return nil
}

// updateDockerfile will add channels staging required for declarative plugin
func updateDockerfile(isLegacyLayout bool) error {
	log.Println("updating Dockerfile to add channels/ directory in the image")
	dockerfile := filepath.Join("Dockerfile")

	controllerPath := "internal/controller/"
	if isLegacyLayout {
		controllerPath = "controllers/"
	}
	// nolint:lll
	err := insertCodeIfDoesNotExist(dockerfile,
		fmt.Sprintf("COPY %s %s", controllerPath, controllerPath),
		"\n# https://github.com/kubernetes-sigs/kubebuilder-declarative-pattern/blob/master/docs/addon/walkthrough/README.md#adding-a-manifest\n# Stage channels and make readable\nCOPY channels/ /channels/\nRUN chmod -R a+rx /channels/")
	if err != nil {
		return err
	}

	err = insertCodeIfDoesNotExist(dockerfile,
		"COPY --from=builder /workspace/manager .",
		"\n# copy channels\nCOPY --from=builder /channels /channels\n")
	if err != nil {
		return err
	}
	return nil
}

// insertCodeIfDoesNotExist insert code if it does not already exists
func insertCodeIfDoesNotExist(filename, target, code string) error {
	// false positive
	// nolint:gosec
	contents, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	idx := strings.Index(string(contents), code)
	if idx != -1 {
		return nil
	}

	return util.InsertCode(filename, target, code)
}
