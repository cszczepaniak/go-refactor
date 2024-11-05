package replace

import (
	"cmp"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"os"

	"github.com/cszczepaniak/go-refactor/internal/analyzeutil"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

func NewTypeReplacer() *analysis.Analyzer {
	var flags struct {
		typeName               string
		replacement            string
		replacementPackageName string
		importAlias            string
	}
	flagSet := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flagSet.StringVar(&flags.typeName, "type", "", "The type to replace. Format is 'github.com/package/path.TypeName'")
	flagSet.StringVar(&flags.replacement, "replacement", "", "The type to replace --type with; takes the same form as --type.")
	flagSet.StringVar(&flags.replacementPackageName, "replacement-package-name", "", "The replacement package name to use.")
	flagSet.StringVar(&flags.importAlias, "import-alias", "", "An optional alias to use when importing the replacement type.")

	return &analysis.Analyzer{
		Name:  "replacetype",
		Doc:   "Replace a type reference with another type.",
		Flags: *flagSet,
		Run: func(pass *analysis.Pass) (interface{}, error) {
			if flags.typeName == "" {
				return nil, errors.New("type is required")
			}

			if flags.replacement == "" {
				return nil, errors.New("replacement is required")
			}

			importer := &analyzeutil.Importer{}
			inspector := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

			typeSpec, err := ParseSymbolSpec(flags.typeName)
			if err != nil {
				return nil, fmt.Errorf("error parsing type: %w", err)
			}

			replacementSpec, err := ParseSymbolSpec(flags.replacement)
			if err != nil {
				return nil, fmt.Errorf("error parsing replacement: %w", err)
			}

			err = doTypeReplacement(
				pass,
				typeSpec,
				replacementSpec,
				flags.replacementPackageName,
				flags.importAlias,
				inspector,
				importer,
			)

			importer.Rewrite(pass)

			return nil, nil
		},
		Requires: []*analysis.Analyzer{
			inspect.Analyzer,
		},
	}
}

func doTypeReplacement(
	pass *analysis.Pass,
	spec SymbolSpec,
	replacement SymbolSpec,
	importName string,
	importAlias string,
	inspector *inspector.Inspector,
	importer *analyzeutil.Importer,
) error {
	var err error
	inspector.WithStack(
		[]ast.Node{&ast.SelectorExpr{}},
		func(n ast.Node, push bool, stack []ast.Node) bool {
			if !push {
				return false
			}

			sel := n.(*ast.SelectorExpr)
			obj := pass.TypesInfo.ObjectOf(sel.Sel)

			switch {
			case spec.matchesTopLevelSymbol(obj):

				var replacementStr string
				if pass.Pkg.Path() == replacement.Pkg {
					replacementStr = replacement.name
				} else {
					importer.Add(pass.Fset, stack[0].(*ast.File), importAlias, replacement.Pkg)
					// Use the import alias if provided, otherwise the name of the import.
					replacementStr = cmp.Or(importAlias, importName) + "." + replacement.name
				}

				err = analyzeutil.ReplaceNode(
					pass,
					sel,
					replacementStr,
				)

				// Whether or not there's an error, there's no need to descend further into a Field.
				return false
			default:
				return true
			}
		},
	)

	return err
}
