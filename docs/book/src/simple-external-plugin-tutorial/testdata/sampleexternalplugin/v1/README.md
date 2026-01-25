# Sample External Plugin

A reference implementation showing how to build external plugins for Kubebuilder.
Adds a Prometheus instance to projects for metrics collection.

## Purpose

This plugin serves as a **comprehensive example** demonstrating:
- External plugin architecture (JSON communication over stdin/stdout)
- Flag parsing and validation
- Reading PROJECT file configuration
- Proper error handling

## What It Does

Scaffolds a Prometheus instance CR that works with the default ServiceMonitor already provided by Kubebuilder:

- `config/prometheus/prometheus.yaml` - Prometheus instance (configurable namespace)
- `config/prometheus/kustomization.yaml` - Resource list with namespace
- `config/default/kustomization_prometheus_patch.yaml` - Setup instructions

**Note**: Kubebuilder already scaffolds `config/prometheus/monitor.yaml` (ServiceMonitor for controller). This plugin adds the Prometheus instance that consumes those metrics.

## Installation

```bash
cd docs/book/src/simple-external-plugin-tutorial/testdata/sampleexternalplugin/v1
make install
```

## Usage

**Initialize new project with Prometheus:**
```bash
kubebuilder init \
  --plugins go/v4,sampleexternalplugin/v1 \
  --domain example.com \
  --repo github.com/example/myoperator \
  --prometheus-namespace monitoring
```

**Add to existing project:**
```bash
kubebuilder edit \
  --plugins sampleexternalplugin/v1 \
  --prometheus-namespace observability
```

## Flags

- `--prometheus-namespace`: Namespace where Prometheus instance will be deployed (default: `monitoring-system`)
  - Must be a valid Kubernetes DNS-1123 label (lowercase alphanumeric + hyphens, max 63 chars)

## How External Plugins Work

External plugins communicate with Kubebuilder via JSON over stdin/stdout:

1. Kubebuilder sends `PluginRequest` (JSON) to plugin via stdin
2. Plugin processes request and returns `PluginResponse` (JSON) via stdout
3. Kubebuilder writes files from response `universe` map to disk

This plugin implements two subcommands:
- `init` - adds Prometheus during project initialization
- `edit` - adds Prometheus to existing projects

See [External Plugins Documentation](https://book.kubebuilder.io/plugins/extending/external-plugins) for details.

## Testing

The plugin uses **Go tests with Ginkgo/Gomega** (same as Kubebuilder internal plugins):

```bash
make test-unit
make test-e2e
make test-plugin
```

**Unit Tests** (`scaffolds/*_test.go`):
- Flag parsing and validation (namespace format)
- Init command with various configs
- Edit command error handling
- PROJECT file reading

**E2E Tests** (`test/e2e/plugin_test.go`):
- Init with custom namespace
- Edit adding Prometheus to existing project
- Namespace validation with invalid input
- Default namespace behavior

## Key Files

- `cmd/cmd.go` - Routes subcommands (init, edit, flags, metadata)
- `scaffolds/init.go` - Init subcommand with detailed flow comments
- `scaffolds/edit.go` - Edit subcommand implementation
- `scaffolds/validation.go` - Input validation helpers
- `internal/test/plugins/prometheus/` - Template generators
- `test/e2e/` - E2E tests
