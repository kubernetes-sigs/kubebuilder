/*
Copyright 2025 The Kubernetes Authors.

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

package update

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"

	"sigs.k8s.io/kubebuilder/v4/pkg/cli/alpha/internal/common"
	"sigs.k8s.io/kubebuilder/v4/pkg/config/store"
)

type releaseResponse struct {
	TagName string `json:"tag_name"`
}

// Prepare resolves version and binary URL details after validation.
// Should be called after Validate().
func (opts *Update) Prepare() error {
	if opts.FromBranch == "" {
		// TODO: Check if is possible to use get to determine the default branch
		log.Warning("No --from-branch specified, using 'main' as default")
		opts.FromBranch = "main"
	}

	path, err := common.GetInputPath("")
	if err != nil {
		return fmt.Errorf("failed to determine project path: %w", err)
	}
	config, err := common.LoadProjectConfig(path)
	if err != nil {
		return fmt.Errorf("failed to load PROJECT config: %w", err)
	}
	opts.FromVersion, err = opts.defineFromVersion(config)
	if err != nil {
		return fmt.Errorf("failed to determine the version to use for the upgrade from: %w", err)
	}
	opts.ToVersion = opts.defineToVersion()
	return nil
}

// defineFromVersion will return the CLI version to be used for the update with the v prefix.
func (opts *Update) defineFromVersion(config store.Store) (string, error) {
	if len(opts.FromBranch) == 0 && len(config.Config().GetCliVersion()) == 0 {
		return "", fmt.Errorf("no version specified in PROJECT file. " +
			"Please use --from-version flag to specify the version to update from")
	}

	if opts.FromVersion != "" {
		if !strings.HasPrefix(opts.FromVersion, "v") {
			return "v" + opts.FromVersion, nil
		}
		return opts.FromVersion, nil
	}
	return "v" + config.Config().GetCliVersion(), nil
}

func (opts *Update) defineToVersion() string {
	if len(opts.ToVersion) != 0 {
		if !strings.HasPrefix(opts.ToVersion, "v") {
			return "v" + opts.ToVersion
		}
		return opts.ToVersion
	}

	opts.ToVersion, _ = fetchLatestRelease()

	return opts.ToVersion
}

func fetchLatestRelease() (string, error) {
	resp, err := http.Get("https://api.github.com/repos/kubernetes-sigs/kubebuilder/releases/latest")
	if err != nil {
		return "", fmt.Errorf("failed to fetch latest release: %w", err)
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Infof("failed to close connection: %s", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var release releaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return release.TagName, nil
}
