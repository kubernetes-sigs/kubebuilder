package merge

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"reflect"
	"strconv"
	"strings"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
	"golang.org/x/tools/go/ast/astutil"
)

func InsertCode(filename, code string) error {
	fset := token.NewFileSet()

	sourceAst, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return err
	}
	mergingAst, err := parser.ParseFile(fset, "", code, parser.ParseComments)
	if err != nil {
		return err
	}

	for _, mImp := range mergingAst.Imports {
		merging := true
		for _, sImp := range sourceAst.Imports {
			if importEqual(sImp, mImp) {
				merging = false
				break
			}
		}
		if merging {
			astutil.AddNamedImport(fset, sourceAst, defaultImportAlias(mImp), mustUnquoteImport(mImp))
		}
	}

	sourceDst, err := decorator.DecorateFile(fset, sourceAst)
	if err != nil {
		return err
	}
	mergingDst, err := decorator.DecorateFile(fset, mergingAst)
	if err != nil {
		return err
	}

	for _, mDecl := range mergingDst.Decls {
		merging := true
		for _, sDecl := range sourceDst.Decls {
			if declEqual(sDecl, mDecl) {
				merging = false
				break
			}
		}
		if merging {
			sourceDst.Decls = append(sourceDst.Decls, mDecl)
		}
	}

	if err = decorator.Print(sourceDst); err != nil {
		return err
	}

	// dst.Fprint(os.Stdout, sourceDst, dst.NotNilFilter)

	return nil
}

func declEqual(src, trg dst.Decl) bool {
	if reflect.TypeOf(src) != reflect.TypeOf(trg) {
		return false
	}
	switch srcType := src.(type) {
	case *dst.GenDecl:
		trgType := trg.(*dst.GenDecl)
		if srcType.Tok != trgType.Tok {
			return false
		}
		switch srcType.Tok {
		case token.IMPORT:
			return true
		case token.TYPE:
			if skipSpecs(srcType.Specs, trgType.Specs) {
				return false
			}
			return srcType.Specs[0].(*dst.TypeSpec).Name.Name == trgType.Specs[0].(*dst.TypeSpec).Name.Name
		case token.VAR:
			if skipSpecs(srcType.Specs, trgType.Specs) {
				return false
			}
			srcVarName := srcType.Specs[0].(*dst.ValueSpec).Names[0].Name
			tgtVarName := trgType.Specs[0].(*dst.ValueSpec).Names[0].Name
			if srcVarName == "_" && tgtVarName == "_" {
				srcVarType := srcType.Specs[0].(*dst.ValueSpec).Type.(*dst.SelectorExpr)
				tgtVarType := trgType.Specs[0].(*dst.ValueSpec).Type.(*dst.SelectorExpr)

				return srcVarType.Sel.Name == tgtVarType.Sel.Name
			}

			return srcVarName == tgtVarName
		default:
			return true
		}
	case *dst.FuncDecl:
		if srcType.Name == nil || trg.(*dst.FuncDecl).Name == nil {
			return false
		}

		return srcType.Name.Name == trg.(*dst.FuncDecl).Name.Name
	}

	return false
}

func skipSpecs(src, tgt []dst.Spec) bool {
	return len(src) == 0 || len(tgt) == 0 || len(src) != len(tgt)
}

func importEqual(src, trg *ast.ImportSpec) bool {
	return src.Path.Value == trg.Path.Value && importAlias(src) == importAlias(trg)
}

func defaultImportAlias(spec *ast.ImportSpec) string {
	if spec.Name == nil {
		return ""
	}

	return spec.Name.Name
}

func importAlias(spec *ast.ImportSpec) string {
	if spec.Name == nil && spec.Path != nil {
		return strings.ReplaceAll(path.Base(mustUnquoteImport(spec)), "-", "")
	}

	return spec.Name.Name
}

func mustUnquoteImport(importSpec *ast.ImportSpec) string {
	res, err := strconv.Unquote(importSpec.Path.Value)
	if err != nil {
		panic(err)
	}

	return res
}
