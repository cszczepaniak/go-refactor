package analyzeutil

import (
	"errors"
	"go/ast"
	"go/token"
	"slices"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ast/astutil"
)

type importModification struct {
	original *ast.File
	mutated  *ast.File
}

type Importer struct {
	filesByName map[string]importModification
}

func (imp *Importer) Add(fset *token.FileSet, f *ast.File, name, path string) {
	if imp.filesByName == nil {
		imp.filesByName = make(map[string]importModification)
	}

	fileName := fset.File(f.Pos()).Name()

	mod, ok := imp.filesByName[fileName]
	if !ok {
		// Inside baseball: we're going to deep clone the fields that AddNamedImport modifies so we
		// can keep the original around.

		clone := *f
		clone.Comments = slices.Clone(f.Comments)
		clone.Decls = slices.Clone(f.Decls)
		clone.Imports = slices.Clone(f.Imports)

		mod = importModification{
			original: f,
			mutated:  &clone,
		}
	}

	astutil.AddNamedImport(fset, mod.mutated, name, path)
	imp.filesByName[fileName] = mod
}

func (imp *Importer) Rewrite(pass *analysis.Pass) error {
	for _, mod := range imp.filesByName {
		if len(mod.original.Decls) == 0 {
			// TODO maybe we should support this? But it's probably impossible to have any
			// replacements in a file with no declarations, I think it would only otherwise be
			// possible to have comments + the package directive.
			return errors.New("attempted to add import to file with no declarations")
		}

		originalDecl, ok := mod.original.Decls[0].(*ast.GenDecl)
		if !ok {
			return errors.New("unimplemented (TODO): we don't yet support there not being an import block")
		}

		if originalDecl.Tok != token.IMPORT {
			return errors.New("unimplemented (TODO): we don't yet support there not being an import block")
		}

		newDecl, ok := mod.mutated.Decls[0].(*ast.GenDecl)
		if !ok || newDecl.Tok != token.IMPORT {
			return errors.New("unexpected error: mutated imports was malformed")
		}

		newDeclStr, err := FormatNode(pass.Fset, newDecl)
		if err != nil {
			return err
		}

		pass.Report(
			analysis.Diagnostic{
				Pos:     originalDecl.Pos(),
				End:     originalDecl.End(),
				Message: "modifying imports",
				SuggestedFixes: []analysis.SuggestedFix{{
					Message: "modifying imports",
					TextEdits: []analysis.TextEdit{{
						Pos:     originalDecl.Pos(),
						End:     originalDecl.End(),
						NewText: []byte(newDeclStr),
					}},
				}},
			},
		)
	}

	return nil
}
