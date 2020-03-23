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

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"sigs.k8s.io/kubebuilder/docs/book/utils/plugin"
)

// argType produces HTML for describing an Argument's type.
func argType(arg *Argument) toHTML {
	if arg.Type == "slice" {
		return span(optionalClasses{"optional": arg.Optional, "slice": true},
			argType(arg.ItemType))
	}

	return span(classes{"optional"},
		Text(arg.Type))
}

// maybeDetails returns HTML describing summary and
// details if present, otherwise returning an empty fragment.
func maybeDetails(help *DetailedHelp) toHTML {
	if help.Summary == "" && help.Details == "" {
		return Fragment{}
	}

	return Fragment{
		details(nil,
			summary(optionalClasses{"no-details": help.Details == ""},
				Text(help.Summary)),
			// NB(directxman12): if we don't wrap with newlines, markdown won't be parsed
			Text(wrapWithNewlines(help.Details)))}
}

// markerTemplate returns HTML describing the documentation for a given marker.
func markerTemplate(marker *MarkerDoc) toHTML {

	// the marker name
	term := dt(classes{"literal", "name"},
		Text(marker.Name))

	// the args summary (displayed in summary mode)
	var fields []toHTML
	for _, field := range marker.Fields {
		fields = append(fields, Fragment{
			dt(optionalClasses{"argument": true, "optional": field.Optional, "literal": true},
				Text(field.Name)),
			dd(optionalClasses{"argument": true, "type": true, "optional": field.Optional},
				argType(&field.Argument)),
		})
	}
	argsDef := dd(classes{"args"},
		dl(classes{"args", "summary"},
			fields...))

	// the argument details (displayed in details mode)
	var args Fragment
	for _, field := range marker.Fields {
		args = append(args, Fragment{
			dt(optionalClasses{"argument": true, "optional": field.Optional, "literal": true},
				Text(field.Name)),
			dd(optionalClasses{"argument": true, "type": true, "optional": field.Optional},
				argType(&field.Argument)),
			dd(classes{"description"},
				maybeDetails(&field.DetailedHelp))})
	}

	// the help (displayed in both modes)
	helpDef := dd(classes{"description"},
		maybeDetails(&marker.DetailedHelp),
		dl(classes{"args"},
			args))

	// the overall wrapping marker (common classes go here to make it easier to select
	// on certain things w/o duplication)
	markerAttrs := attrs{
		optionalClasses{
			"marker":     true,
			"deprecated": marker.DeprecatedInFavorOf != nil,
			"anonymous":  marker.Anonymous(),
		},
		dataAttr{Name: "target", Value: marker.Target},
	}
	if marker.DeprecatedInFavorOf != nil {
		markerAttrs = append(markerAttrs, dataAttr{Name: "deprecated", Value: *marker.DeprecatedInFavorOf})
	}
	return div(markerAttrs,
		term, argsDef, helpDef)
}

// MarkerDocs is a plugin that autogenerates documentation
// for markers known to controller-gen.  Generated pages
// will be added to locations marked `{{#markerdocs category name}}`.
// This allows us to put additional documentation in each category.
type MarkerDocs struct {
	// Args contains the arguments to pass to controller-gen to get
	// marker help JSON output.  Each key is a prefix to apply to
	// category names (for disambiguation), and each value is an invocation
	// of controller-gen.
	Args map[string][]string
}

// SupportsOutput implements plugin.Plugin
func (MarkerDocs) SupportsOutput(_ string) bool { return true }

// Process implements plugin.Plugin
func (p MarkerDocs) Process(input *plugin.Input) error {
	markerDocs, err := p.getMarkerDocs()
	if err != nil {
		return fmt.Errorf("unable to fetch marker docs: %v", err)
	}

	// first, find all categories...
	markersByCategory := make(map[string][]MarkerDoc)
	for _, cat := range markerDocs {
		if cat.Category == "" {
			// skip un-named categories, which are intended to be hidden
			// (e.g. all the per-generate idendical output rules)
			continue
		}
		markersByCategory[cat.Category] = cat.Markers
	}

	usedCategories := make(map[string]struct{}, len(markersByCategory))

	// NB(directxman12): we use existing pages instead of generating new ones so that we can add additional
	// content to the pages (for instance, additional description of the category).

	// ...then, go through the book, finding all instances of `{{#markerdocs <category>}}` and replacing them
	// with the appropriate docs ...
	err = plugin.EachCommand(&input.Book, "markerdocs", func(chapter *plugin.BookChapter, category string) (string, error) {
		category = strings.TrimSpace(category)
		markers, knownCategory := markersByCategory[category]
		if !knownCategory {
			return "", fmt.Errorf("unknown category %q", category)
		}

		// HTML5 says that any characters are valid in ID except for space,
		// but may not be empty (which we prevent by skipping un-named categories):
		// https://www.w3.org/TR/html52/dom.html#element-attrdef-global-id
		categoryAlias := strings.ReplaceAll(category, " ", "-")

		content := new(strings.Builder)

		// NB(directxman12): wrap this in a div to prevent the markdown processor from inserting extra paragraphs
		_, err := fmt.Fprintf(content, "<div><input checked type=\"checkbox\" class=\"markers-summarize\" id=\"markers-summarize-%[1]s\"></input><label class=\"markers-summarize\" for=\"markers-summarize-%[1]s\">Show Detailed Argument Help</label><dl class=\"markers\">", categoryAlias)
		if err != nil {
			return "", fmt.Errorf("unable to render marker documentation summary: %v", err)
		}

		// write the markers
		for _, marker := range markers {
			if err := markerTemplate(&marker).WriteHTML(content); err != nil {
				return "", fmt.Errorf("unable to render documentation for marker %q: %v", marker.Name, err)
			}
		}

		if _, err = fmt.Fprintf(content, "</dl></div>"); err != nil {
			return "", fmt.Errorf("unable to render marker documentation: %v", err)
		}

		usedCategories[category] = struct{}{}

		return content.String(), nil
	})
	if err != nil {
		return err
	}

	// ... and finally make sure we didn't miss any
	if len(usedCategories) != len(markersByCategory) {
		unusedCategories := make([]string, 0, len(markersByCategory)-len(usedCategories))
		for cat := range markersByCategory {
			if _, ok := usedCategories[cat]; !ok {
				unusedCategories = append(unusedCategories, cat)
			}
		}
		return fmt.Errorf("unused categories %v", unusedCategories)
	}

	return nil
}

// wrapWithNewlines ensures that we begin and end with a newline character.
// this is important to ensure that markdown is parsed inside of details elements.
func wrapWithNewlines(src string) string {
	if len(src) < 4 {
		return src
	}
	if src[0] != '\n' {
		src = "\n" + src
	}
	if src[1] != '\n' {
		src = "\n" + src
	}
	if src[len(src)-1] != '\n' {
		src = src + "\n"
	}
	if src[len(src)-2] != '\n' {
		src = src + "\n"
	}
	return src
}

// getMarkerDocs fetches marker documentation from controller-gen
func (p MarkerDocs) getMarkerDocs() ([]CategoryDoc, error) {
	var res []CategoryDoc
	for categoryPrefix, args := range p.Args {
		cmd := exec.Command("controller-gen", args...)
		outRaw, err := cmd.Output()
		if err != nil {
			return nil, err
		}

		var invocationRes []CategoryDoc
		if err := json.Unmarshal(outRaw, &invocationRes); err != nil {
			return nil, err
		}

		for i, category := range invocationRes {
			// leave empty categories as-is, so that they're skipped
			if category.Category == "" {
				continue
			}
			invocationRes[i].Category = categoryPrefix + category.Category
		}

		res = append(res, invocationRes...)
	}

	return res, nil
}

func main() {
	if err := plugin.Run(MarkerDocs{
		Args: map[string][]string{
			// marker args
			"": []string{"-wwww", "crd", "webhook", "rbac:roleName=cheddar" /* role name doesn't mean anything here */, "object", "schemapatch:manifests=."},
			// cli options args
			"CLI: ": []string{"-hhhh"},
		},
	}, os.Stdin, os.Stdout, os.Args[1:]...); err != nil {
		log.Fatal(err.Error())
	}
}
