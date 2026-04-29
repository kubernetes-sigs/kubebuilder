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

package github

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &AutoUpdate{}

// AutoUpdate scaffolds the GitHub Action to lint the project
type AutoUpdate struct {
	machinery.TemplateMixin
	machinery.BoilerplateMixin

	// UseGHModels indicates whether to enable GitHub Models AI summary
	UseGHModels bool

	// OpenGHIssue indicates whether to create GitHub Issues
	OpenGHIssue bool

	// OpenGHPR indicates whether to create GitHub Pull Requests
	OpenGHPR bool
}

// SetTemplateDefaults implements machinery.Template
func (f *AutoUpdate) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join(".github", "workflows", "auto_update.yml")
	}

	f.TemplateBody = autoUpdateTemplate
	// Always overwrite to keep workflow in sync with configuration
	f.IfExistsAction = machinery.OverwriteFile

	return nil
}

const autoUpdateTemplate = `name: Auto Update

# The 'kubebuilder alpha update' command requires write access to the repository to create a branch
# with the update files and optionally create a pull request (--open-gh-pr) and/or issue (--open-gh-issue).
# The branch created will be named in the format kubebuilder-update-from-<from-version>-to-<to-version> by default.
# To protect your codebase, please ensure that you have branch protection rules configured for your
# main branches. This will guarantee that no one can bypass a review and push directly to a branch like 'main'.
permissions: {}

on:
  workflow_dispatch:
  schedule:
    - cron: "0 0 * * 2" # Every Tuesday at 00:00 UTC

jobs:
  auto-update:
    permissions:
      contents: write  # Create and push the update branch
{{- if .OpenGHIssue }}
      issues: write  # Create GitHub Issue notification
{{- end }}
{{- if .OpenGHPR }}
      pull-requests: write  # Create Pull Request
{{- end }}
{{- if .UseGHModels }}
      models: read  # Use GitHub Models
{{- end }}
    runs-on: ubuntu-latest
    env:
      GH_TOKEN: {{ "${{ secrets.GITHUB_TOKEN }}" }}

    # Checkout the repository.
    steps:
    - name: Checkout repository
      uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v6.0.2
      with:
        token: {{ "${{ secrets.GITHUB_TOKEN }}" }}
        fetch-depth: 0
        persist-credentials: false

    # Configure Git to create commits with the GitHub Actions bot.
    - name: Configure Git
      run: |
        git config --global user.name "github-actions[bot]"
        git config --global user.email "github-actions[bot]@users.noreply.github.com"

    # Set up Go environment.
    - name: Set up Go
      uses: actions/setup-go@4b73464bb391d4059bd26b0524d20df3927bd417 # v6.3.0
      with:
        go-version: stable

    # Install Kubebuilder.
    - name: Install Kubebuilder
      run: |
        curl -L -o kubebuilder "https://go.kubebuilder.io/dl/latest/$(go env GOOS)/$(go env GOARCH)"
        chmod +x kubebuilder
        sudo mv kubebuilder /usr/local/bin/
        kubebuilder version
{{ if .UseGHModels }}
    # Install Models extension for GitHub CLI.
    - name: Install gh-models extension
      run: |
        gh extension install github/gh-models --force
        gh models --help >/dev/null
{{ end }}
    # Run the Kubebuilder alpha update command.
    # More info: https://kubebuilder.io/reference/commands/alpha_update
    - name: Run kubebuilder alpha update
      # Executes the update command with specified flags.
      # --force: Completes the merge even if conflicts occur, leaving conflict markers.
      # --push: Automatically pushes the resulting output branch to the 'origin' remote.
      # --restore-path: Preserves specified paths (e.g., CI workflow files) when squashing.{{ if .OpenGHIssue }}
      # --open-gh-issue: Creates a GitHub Issue.{{ end }}{{ if .OpenGHPR }}
      # --open-gh-pr: Creates a GitHub Pull Request directly for review.{{ if .UseGHModels }}
      # --use-gh-models: Adds an AI summary to the PR description.{{ else }}
      #
      # WARNING: This workflow does not use GitHub Models AI summary by default.
      # To enable AI-generated summaries, you need permissions to use GitHub Models.
      # If you have the required permissions, re-run:
      #   kubebuilder edit --plugins="autoupdate/v1-alpha" --use-gh-models{{ end }}{{ end }}
      run: |
        kubebuilder alpha update \
          --force \
          --push \
          --restore-path .github/workflows{{ if .OpenGHIssue }} \
          --open-gh-issue{{ end }}{{ if .OpenGHPR }} \
          --open-gh-pr{{ end }}{{ if .UseGHModels }} \
          --use-gh-models{{ end }}
`
