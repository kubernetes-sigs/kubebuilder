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

package util

import (
	"fmt"
	"regexp"
)

// The following code came from "k8s.io/apimachinery/pkg/util/validation/validation.go"
// If be required the usage of more funcs from this then please replace it for the import
// ---------------------------------------

const (
	qnameCharFmt string = "[a-z0-9]([-a-z0-9]*[a-z0-9])?"
	// The value is 56 because it will be contact with "-system" = 63
	qualifiedNameMaxLength int = 56
)

var qualifiedNameRegexp = regexp.MustCompile("^" + qnameCharFmt + "$")

//IsValidName used to check the name of the project
func IsValidName(value string) []string {
	var errs []string
	if len(value) > qualifiedNameMaxLength {
		errs = append(errs, MaxLenError(qualifiedNameMaxLength))
	}
	if !qualifiedNameRegexp.MatchString(value) {
		errs = append(errs, RegexError("invalid value for project name", qnameCharFmt))
	}
	return errs
}

// RegexError returns a string explanation of a regex validation failure.
func RegexError(msg string, fmt string, examples ...string) string {
	if len(examples) == 0 {
		return msg + " (regex used for validation is '" + fmt + "')"
	}
	msg += " (e.g. "
	for i := range examples {
		if i > 0 {
			msg += " or "
		}
		msg += "'" + examples[i] + "', "
	}
	msg += "regex used for validation is '" + fmt + "')"
	return msg
}

// MaxLenError returns a string explanation of a "string too long" validation
// failure.
func MaxLenError(length int) string {
	return fmt.Sprintf("must be no more than %d characters", length)
}
