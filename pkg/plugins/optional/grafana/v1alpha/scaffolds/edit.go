/*
Copyright 2022 The Kubernetes Authors.

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

package scaffolds

import (
	"fmt"
	"io"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/grafana/v1alpha/scaffolds/internal/templates"

	"sigs.k8s.io/yaml"
)

var _ plugins.Scaffolder = &editScaffolder{}

const configFilePath = "grafana/custom-metrics/config.yaml"

type editScaffolder struct {
	// fs is the filesystem that will be used by the scaffolder
	fs machinery.Filesystem
}

// NewEditScaffolder returns a new Scaffolder for project edition operations
func NewEditScaffolder() plugins.Scaffolder {
	return &editScaffolder{}
}

// InjectFS implements cmdutil.Scaffolder
func (s *editScaffolder) InjectFS(fs machinery.Filesystem) {
	s.fs = fs
}

func fileExist(configFilePath string) bool {
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		return false
	}
	return true
}

func loadConfig(configPath string) ([]templates.CustomMetricItem, error) {
	if !fileExist(configPath) {
		return nil, nil
	}

	// nolint:gosec
	f, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("error loading plugin config: %w", err)
	}

	items, err := configReader(f)

	if err := f.Close(); err != nil {
		return nil, fmt.Errorf("could not close config.yaml: %w", err)
	}

	return items, err
}

func configReader(reader io.Reader) ([]templates.CustomMetricItem, error) {
	yamlFile, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	config := templates.CustomMetricsConfig{}

	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		return nil, err
	}

	validatedMetricItems := validateCustomMetricItems(config.CustomMetrics)

	return validatedMetricItems, nil
}

func validateCustomMetricItems(rawItems []templates.CustomMetricItem) []templates.CustomMetricItem {
	// 1. Filter items of missing `Metric` or `Type`
	filterResult := []templates.CustomMetricItem{}
	for _, item := range rawItems {
		if hasFields(item) {
			filterResult = append(filterResult, item)
		}
	}

	// 2. Fill Expr and Unit if missing
	validatedItems := make([]templates.CustomMetricItem, len(filterResult))
	for i, item := range filterResult {
		item = fillMissingExpr(item)
		validatedItems[i] = fillMissingUnit(item)
	}

	return validatedItems
}

func hasFields(item templates.CustomMetricItem) bool {
	// If `Expr` exists, return true
	if item.Expr != "" {
		return true
	}

	// If `Metric` & valid `Type` exists, return true
	metricType := strings.ToLower(item.Type)
	if item.Metric != "" && (metricType == "counter" || metricType == "gauge" || metricType == "histogram") {
		return true
	}

	return false
}

// TODO: Prom_ql exprs can improved to be more pratical and applicable
func fillMissingExpr(item templates.CustomMetricItem) templates.CustomMetricItem {
	if item.Expr == "" {
		switch strings.ToLower(item.Type) {
		case "counter":
			item.Expr = "sum(rate(" + item.Metric + `{job=\"$job\", namespace=\"$namespace\"}[5m])) by (instance, pod)`
		case "histogram":
			// nolint: lll
			item.Expr = "histogram_quantile(0.90, sum by(instance, le) (rate(" + item.Metric + `{job=\"$job\", namespace=\"$namespace\"}[5m])))`
		default: // gauge
			item.Expr = item.Metric
		}
	}
	return item
}

func fillMissingUnit(item templates.CustomMetricItem) templates.CustomMetricItem {
	if item.Unit == "" {
		name := strings.ToLower(item.Metric)
		item.Unit = "none"
		if strings.Contains(name, "second") || strings.Contains(name, "duration") {
			item.Unit = "s"
		} else if strings.Contains(name, "byte") {
			item.Unit = "bytes"
		} else if strings.Contains(name, "ratio") {
			item.Unit = "percent"
		}
	}
	return item
}

// Scaffold implements cmdutil.Scaffolder
func (s *editScaffolder) Scaffold() error {
	log.Println("Generating Grafana manifests to visualize controller status...")

	// Initialize the machinery.Scaffold that will write the files to disk
	scaffold := machinery.NewScaffold(s.fs)

	configPath := string(configFilePath)

	var templatesBuilder = []machinery.Builder{
		&templates.RuntimeManifest{},
		&templates.ResourcesManifest{},
		&templates.CustomMetricsConfigManifest{ConfigPath: configPath},
	}

	configItems, err := loadConfig(configPath)
	if err == nil && len(configItems) > 0 {
		templatesBuilder = append(templatesBuilder, &templates.CustomMetricsDashManifest{Items: configItems})
	} else if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error on scaffolding manifest for custom metris:\n%v", err)
	}

	return scaffold.Execute(templatesBuilder...)
}
