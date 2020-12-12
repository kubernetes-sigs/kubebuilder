/*
Copyright 2020 The Kubernetes Authors.

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

package resource

import (
	"path"
	"strings"

	"github.com/gobuffalo/flect"
)

// safeImport returns a cleaned version of the provided string that can be used for imports
func safeImport(unsafe string) string {
	safe := unsafe

	// Remove dashes and dots
	safe = strings.Replace(safe, "-", "", -1)
	safe = strings.Replace(safe, ".", "", -1)

	return safe
}

// LocalPath returns the default path
func LocalPath(repo, group, version string, multiGroup bool) string {
	if multiGroup {
		if group != "" {
			return path.Join(repo, "apis", group, version)
		}
		return path.Join(repo, "apis", version)
	}
	return path.Join(repo, "api", version)
}

// RegularPlural returns a default plural form when none was specified
func RegularPlural(kind string) string {
	return flect.Pluralize(strings.ToLower(kind))
}
