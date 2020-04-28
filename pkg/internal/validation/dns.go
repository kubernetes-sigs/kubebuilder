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

package validation

import (
	"fmt"
	"regexp"
)

// This file's code was modified from "k8s.io/apimachinery/pkg/util/validation"
// to avoid package dependencies. In case of additional functionality from
// "k8s.io/apimachinery" is needed, re-consider whether to add the dependency.

const (
	dns1123LabelFmt     string = "[a-z0-9]([-a-z0-9]*[a-z0-9])?"
	dns1123SubdomainFmt string = dns1123LabelFmt + "(\\." + dns1123LabelFmt + ")*"
	dns1035LabelFmt     string = "[a-z]([-a-z0-9]*[a-z0-9])?"
)

type dnsValidationConfig struct {
	format   string
	maxLen   int
	re       *regexp.Regexp
	errMsg   string
	examples []string
}

var dns1123LabelConfig = dnsValidationConfig{
	format:   dns1123LabelFmt,
	maxLen:   56, // = 63 - len("-system")
	re:       regexp.MustCompile("^" + dns1123LabelFmt + "$"),
	errMsg:   "a DNS-1123 label must consist of lower case alphanumeric characters or '-'",
	examples: []string{"example.com"},
}

var dns1123SubdomainConfig = dnsValidationConfig{
	format: dns1123SubdomainFmt,
	maxLen: 253, // a subdomain's max length in DNS (RFC 1123).
	re:     regexp.MustCompile("^" + dns1123SubdomainFmt + "$"),
	errMsg: "a DNS-1123 subdomain must consist of lower case alphanumeric characters, " +
		"'-' or '.', and must start and end with an alphanumeric character",
	examples: []string{"my-name", "abc-123"},
}

var dns1035LabelConfig = dnsValidationConfig{
	format: dns1035LabelFmt,
	maxLen: 63, // a label's max length in DNS (RFC 1035).
	re:     regexp.MustCompile("^" + dns1035LabelFmt + "$"),
	errMsg: "a DNS-1035 label must consist of lower case alphanumeric characters or '-', " +
		"start with an alphabetic character, and end with an alphanumeric character",
	examples: []string{"my-name", "123-abc"},
}

func (c dnsValidationConfig) check(value string) (errs []string) {
	if len(value) > c.maxLen {
		errs = append(errs, maxLenError(c.maxLen))
	}
	if !c.re.MatchString(value) {
		errs = append(errs, regexError(c.errMsg, c.format, c.examples...))
	}
	return errs
}

// IsDNS1123Subdomain tests for a string that conforms to the definition of a
// subdomain in DNS (RFC 1123).
func IsDNS1123Subdomain(value string) []string {
	return dns1123SubdomainConfig.check(value)
}

// IsDNS1123Label tests for a string that conforms to the definition of a label in DNS (RFC 1123).
func IsDNS1123Label(value string) []string {
	return dns1123LabelConfig.check(value)
}

// IsDNS1035Label tests for a string that conforms to the definition of a label in DNS (RFC 1035).
func IsDNS1035Label(value string) []string {
	return dns1035LabelConfig.check(value)
}

// maxLenError returns a string explanation of a "string too long" validation
// failure.
func maxLenError(length int) string {
	return fmt.Sprintf("must be no more than %d characters", length)
}

// regexError returns a string explanation of a regex validation failure.
func regexError(msg string, fmt string, examples ...string) string {
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
