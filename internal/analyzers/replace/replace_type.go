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
		[]ast.Node{&ast.SelectorExpr{}, &ast.Ident{}},
		func(n ast.Node, push bool, stack []ast.Node) bool {
			if !push {
				return false
			}

			var name *ast.Ident
			switch n := n.(type) {
			case *ast.SelectorExpr:
				name = n.Sel
			case *ast.Ident:
				// If we have an identifier, let's make sure we're not the name of a type spec. If
				// we are, we shouldn't replace this one because our goal isn't to remove the old
				// type spec, just to replace all of its usages.
				if _, ok := stack[len(stack)-2].(*ast.TypeSpec); ok {
					return true
				}

				name = n
			default:
				panic("unreachable")
			}

			obj := pass.TypesInfo.ObjectOf(name)

			switch {
			case spec.matchesTopLevelSymbol(obj):

				var replacementStr string
				if pass.Pkg.Path() == replacement.Pkg {
					replacementStr = replacement.name
				} else {
					addedName := importer.Add(pass.Fset, stack[0].(*ast.File), importAlias, replacement.Pkg)
					// Use the import alias if provided, otherwise the name of the import.
					replacementStr = cmp.Or(addedName, importName) + "." + replacement.name
				}

				err = analyzeutil.ReplaceNode(
					pass,
					n,
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
