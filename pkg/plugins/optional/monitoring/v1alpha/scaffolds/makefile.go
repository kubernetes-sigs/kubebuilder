package scaffolds

import (
	"fmt"
	"os"

	"github.com/spf13/afero"

	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
)

// Update Makefile to generate the Prometheus metrics docs
func updateMakefile(fs machinery.Filesystem) error {
	filename := "Makefile"

	makefileBytes, err := afero.ReadFile(fs.FS, filename)
	if err != nil {
		return err
	}

	makefileBytes = append(makefileBytes, []byte(makefileBundleVarFragment)...)

	var mode os.FileMode = 0644
	if info, err := fs.FS.Stat(filename); err == nil {
		mode = info.Mode()
	}
	if err = afero.WriteFile(fs.FS, filename, makefileBytes, mode); err != nil {
		return fmt.Errorf("error updating %s: %w", filename, err)
	}

	return nil
}

const makefileBundleVarFragment = `
##@ Create the metrics doc file

.PHONY: generate-metricsdocs
generate-metricsdocs: build-metricsdocs
	mkdir -p $(shell pwd)/docs/monitoring
	_out/metricsdocs > docs/monitoring/metrics.md

.PHONY: build-metricsdocs
build-metricsdocs:
	go build -ldflags="${LDFLAGS}" -o _out/metricsdocs ./monitoring/tools/metricsdocs.go
`
