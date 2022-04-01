/*
Copyright 2019 Joe Lanford.

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

package diff

import (
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/exp/apidiff"
)

type Diff struct {
	selfIncompatible    bool
	importsIncompatible bool
	selfReports         map[string]apidiff.Report
	importsReports      map[string]apidiff.Report
}

func (d *Diff) IsCompatible() bool {
	return !d.selfIncompatible && !d.importsIncompatible
}

func (d *Diff) PrintReports(printCompatible bool) {
	w := &prefixWriter{prefix: "  ", w: os.Stdout}
	for pkg, report := range d.selfReports {
		writeReport(w, pkg, report, printCompatible)
	}

	for pkg, report := range d.importsReports {
		writeReport(w, pkg, report, printCompatible)
	}
}

type prefixWriter struct {
	prefix string
	w      io.Writer
}

func (w prefixWriter) Write(d []byte) (int, error) {
	toWrite := fmt.Sprintf("%s%s\n", w.prefix, strings.TrimSpace(strings.ReplaceAll(string(d), "\n", "\n"+w.prefix)))
	return w.w.Write([]byte(toWrite))
}

func writeReport(w io.Writer, name string, report apidiff.Report, printCompatible bool) error {
	var (
		hasIncompatible bool
		hasCompatible   bool
	)
	for _, c := range report.Changes {
		if !c.Compatible {
			hasIncompatible = true
		} else {
			hasCompatible = true
		}
	}

	if hasIncompatible || (printCompatible && hasCompatible) {
		if _, err := fmt.Fprintf(os.Stdout, "\n%s\n", name); err != nil {
			return err
		}
	}
	if hasIncompatible {
		if err := report.TextIncompatible(w, true); err != nil {
			return err
		}
	}
	if printCompatible && hasCompatible {
		if err := report.TextCompatible(w); err != nil {
			return err
		}
	}
	return nil
}
