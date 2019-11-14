/*
Copyright 2019 The Kubernetes Authors.

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

package main

// these should be kept in sync with the output from controller-tools

type DetailedHelp struct {
	Summary string `json:"summary"`
	Details string `json:"details"`
}

type Argument struct {
	Type     string    `json:"type"`
	Optional bool      `json:"optional"`
	ItemType *Argument `json:"itemType"`
}

type FieldHelp struct {
	// definition
	Name     string `json:"name"`
	Argument `json:",inline"`

	// help

	DetailedHelp `json:",inline"`
}

type MarkerDoc struct {
	// definition

	Name   string `json:"name"`
	Target string `json:"target"`

	// help

	DetailedHelp        `json:",inline"`
	Category            string      `json:"category"`
	DeprecatedInFavorOf *string     `json:"deprecatedInFavorOf"`
	Fields              []FieldHelp `json:"fields"`
}

type CategoryDoc struct {
	Category string      `json:"category"`
	Markers  []MarkerDoc `json:"markers"`
}

func (m MarkerDoc) Anonymous() bool {
	return len(m.Fields) == 1 && m.Fields[0].Name == ""
}
