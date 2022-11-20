package scaffolds

import (
	"fmt"

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin/util"
)

// Update main.go to register Prometheus metrics
func updateMain(fs machinery.Filesystem, config config.Config) error {
	file, err := fs.FS.Open("main.go")
	if err != nil {
		return err
	}
	defer file.Close()

	importFragment := fmt.Sprintf(importVarFragment, config.GetRepository())
	if err = util.InsertCode(file.Name(), "//+kubebuilder:scaffold:imports", importFragment); err != nil {
		return err
	}

	if err = util.InsertCode(file.Name(), "func init() {", registerVarFragment); err != nil {
		return err
	}

	return nil
}

const (
	importVarFragment = `

	"%s/monitoring/metrics"`

	registerVarFragment = `
	metrics.RegisterMetrics()
`
)
