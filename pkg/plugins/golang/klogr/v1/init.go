/*
Copyright 2021 The Kubernetes Authors.

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

package v1

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"regexp"
	"strconv"
	"strings"

	"github.com/spf13/afero"
	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"
)

var _ plugin.InitSubcommand = &initSubcommand{}

type initSubcommand struct {
	config config.Config
}

func (p *initSubcommand) InjectConfig(c config.Config) error {
	p.config = c

	return nil
}

func (p *initSubcommand) Scaffold(fs machinery.Filesystem) error {
	if err := p.rewriteMain(fs); err != nil {
		return fmt.Errorf("error rewriting main.go to use klogr: %w", err)
	}
	return nil
}

func (p *initSubcommand) rewriteMain(fs machinery.Filesystem) error {
	filePath := "main.go"

	b, err := afero.ReadFile(fs.FS, filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	c := &goCode{
		content: string(b),
	}

	ci := changeImports{
		AddImports:    []string{"k8s.io/klog/v2", "k8s.io/klog/v2/klogr"},
		RemoveImports: []string{"sigs.k8s.io/controller-runtime/pkg/log/zap"},
	}
	if err := ci.Apply(c); err != nil {
		return err
	}

	if err := c.ReplaceCode(
		"opts := zap.Options{ Development: true, } opts.BindFlags(flag.CommandLine)",
		"klog.InitFlags(nil)"); err != nil {
		return err
	}

	if err := c.ReplaceCode(
		"ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))",
		"ctrl.SetLogger(klogr.New())"); err != nil {
		return err
	}

	if err := afero.WriteFile(fs.FS, filePath, []byte(c.content), 0); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// goCode is a simple wrapper around a file containing go source code.
type goCode struct {
	content string
}

// ReplaceCode performs a simple code replacement.
// It is whitespace tolerant, but otherwise not very smart.
// If the "find" string is not found, it therefore returns an error.
func (c *goCode) ReplaceCode(find, replace string) error {
	r := regexp.QuoteMeta(find)
	r = strings.ReplaceAll(r, " ", "\\s*") // ignore whitespace
	r = "(?m)" + r                         // multiline
	compiled, err := regexp.Compile(r)
	if err != nil {
		return fmt.Errorf("failed to compile regexp %q: %w", r, err)
	}
	matches := compiled.FindAllString(c.content, -1)
	if len(matches) == 0 {
		return fmt.Errorf("failed to match %q", find)
	}
	c.content = compiled.ReplaceAllString(c.content, replace)
	return nil
}

// changeImports is a code transformation that adds and removes import statements.
type changeImports struct {
	AddImports    []string
	RemoveImports []string
}

// Apply adds and removes the import statements to the provided goCode.
func (o *changeImports) Apply(c *goCode) error {
	fileSet := token.NewFileSet()

	file, err := parser.ParseFile(fileSet, "", c.content, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse go code: %w", err)
	}

	for _, decl := range file.Decls {
		switch decl := decl.(type) {
		case *ast.GenDecl:
			if decl.Tok == token.IMPORT {
				var newSpecs []ast.Spec

				for _, spec := range decl.Specs {
					shouldRemove, err := o.shouldRemoveImport(spec)
					if err != nil {
						return err
					}
					if !shouldRemove {
						newSpecs = append(newSpecs, spec)
					}
				}
				for _, addImport := range o.AddImports {
					newSpecs = append(newSpecs, &ast.ImportSpec{
						Path: &ast.BasicLit{
							Value: strconv.Quote(addImport),
						},
					})
				}
				decl.Specs = newSpecs
			}
		}
	}

	var b bytes.Buffer
	if err := printer.Fprint(&b, fileSet, file); err != nil {
		return fmt.Errorf("failed to serialize AST: %w", err)
	}

	c.content = b.String()

	return nil
}

func (o *changeImports) shouldRemoveImport(spec ast.Spec) (bool, error) {
	if len(o.RemoveImports) == 0 {
		return false, nil
	}

	switch spec := spec.(type) {
	case *ast.ImportSpec:
		if spec.Path == nil {
			return false, nil
		}
		s, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			return false, fmt.Errorf("unable to decode import %q", spec.Path.Value)
		}

		for _, removeImport := range o.RemoveImports {
			if s == removeImport {
				return true, nil
			}
		}
		return false, nil

	default:
		return false, fmt.Errorf("unknown type %T", spec)
	}
}
