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
	"fmt"
	"html"
	"io"
	"strings"
)

// NB(directxman12): we use this instead of templates to avoid
// weird issues with whitespace in elements rendered as inline.
// Writing with templates was getting tricky to do without
// compromising readability.
//
// This isn't an amazing solution, but it's good enough™

// toHTML knows how to write itself as HTML to an output.
type toHTML interface {
	// WriteHTML writes this as HTML to the given Writer.
	WriteHTML(io.Writer) error
}

// Text is a chunk of text in an HTML doc.
type Text string

// WriteHTML writes the string as HTML to the given Writer
func (t Text) WriteHTML(w io.Writer) error {
	_, err := io.WriteString(w, html.EscapeString(string(t)))
	return err
}

// Tag is some tag with contents and attributes in an HTML doc.
type Tag struct {
	Name     string
	Attrs    Attrs
	Children []toHTML
}

// WriteHTML writes the tag as HTML to the given Writer
func (t Tag) WriteHTML(w io.Writer) error {
	attrsOut := ""
	if t.Attrs != nil {
		attrsOut = t.Attrs.ToAttrs()
	}
	if _, err := fmt.Fprintf(w, "<%s %s>", t.Name, attrsOut); err != nil {
		return err
	}

	for _, child := range t.Children {
		if err := child.WriteHTML(w); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(w, "</%s>", t.Name); err != nil {
		return err
	}

	return nil
}

// Fragment is some series of tags, text, etc in an HTML doc.
type Fragment []toHTML

// WriteHTML writes the fragment as HTML to the given Writer
func (f Fragment) WriteHTML(w io.Writer) error {
	for _, item := range f {
		if err := item.WriteHTML(w); err != nil {
			return err
		}
	}
	return nil
}

// Attrs knows how to convert itself to HTML attributes.
type Attrs interface {
	// ToAttrs returns `key1="value1" key2="value2"` etc to be placed into an HTML tag.
	ToAttrs() string
}

// classes sets the class attribute to these class names.
type classes []string

// ToAttrs implements Attrs
func (c classes) ToAttrs() string { return fmt.Sprintf("class=%q", strings.Join(c, " ")) }

// optionalClasses sets the the class attribute to these class names, if their values are true.
type optionalClasses map[string]bool

// ToAttrs implements Attrs
func (c optionalClasses) ToAttrs() string {
	actualClasses := make([]string, 0, len(c))
	for class, active := range c {
		if active {
			actualClasses = append(actualClasses, class)
		}
	}
	return classes(actualClasses).ToAttrs()
}

// attrs joins together one or more Attrs.
type attrs []Attrs

// ToAttrs implements Attrs
func (a attrs) ToAttrs() string {
	parts := make([]string, len(a))
	for i, attr := range a {
		parts[i] = attr.ToAttrs()
	}
	return strings.Join(parts, " ")
}

// dataAttr represents some `data-*` attribute.
type dataAttr struct {
	Name  string
	Value string
}

// ToAttrs implements Attrs
func (d dataAttr) ToAttrs() string {
	return fmt.Sprintf("data-%s=%q", d.Name, d.Value)
}

// makeTag produces a function that makes tags of the given
// type.
func makeTag(name string) func(Attrs, ...toHTML) Tag {
	return func(attrs Attrs, children ...toHTML) Tag {
		return Tag{
			Name:     name,
			Attrs:    attrs,
			Children: children,
		}
	}
}

var (
	dd      = makeTag("dd")
	dt      = makeTag("dt")
	dl      = makeTag("dl")
	details = makeTag("details")
	summary = makeTag("summary")
	span    = makeTag("span")
	div     = makeTag("div")
)
