package replace

import (
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

func NewFuncReplacer() *analysis.Analyzer {
	var flags struct {
		function    string
		replacement string
	}
	flagSet := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flagSet.StringVar(&flags.function, "func", "", "The function to replace. Format is 'github.com/package/path.FunctionName'")
	flagSet.StringVar(&flags.replacement, "replacement", "", "The replacement string. Placeholders are available (like $arg0).")

	return &analysis.Analyzer{
		Name:  "replacecall",
		Doc:   "Replace a function call with something else.",
		Flags: *flagSet,
		Run: func(pass *analysis.Pass) (interface{}, error) {
			if flags.function == "" {
				return nil, errors.New("func must be provided")
			}

			importer := &analyzeutil.Importer{}
			inspector := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

			spec, err := ParseSymbolSpec(flags.function)
			if err != nil {
				return nil, fmt.Errorf("error parsing func: %w", err)
			}

			r, err := parseReplacement(flags.replacement)
			if err != nil {
				return nil, err
			}

			err = doFunctionReplacement(pass, spec, inspector, importer, r)
			if err != nil {
				return nil, err
			}

			importer.Rewrite(pass)

			return nil, nil
		},
		Requires: []*analysis.Analyzer{
			inspect.Analyzer,
		},
	}
}

func doFunctionReplacement(
	pass *analysis.Pass,
	parsedFunc SymbolSpec,
	inspector *inspector.Inspector,
	importer *analyzeutil.Importer,
	r parsedReplacement,
) error {
	var err error
	inspector.WithStack(
		[]ast.Node{&ast.CallExpr{}},
		func(n ast.Node, push bool, stack []ast.Node) bool {
			if !push {
				return false
			}

			callExpr := n.(*ast.CallExpr)

			var name *ast.Ident
			switch fn := callExpr.Fun.(type) {
			case *ast.Ident:
				name = fn
			case *ast.SelectorExpr:
				name = fn.Sel
			default:
				return true
			}

			obj := pass.TypesInfo.ObjectOf(name)

			switch {
			case parsedFunc.matchesTopLevelSymbol(obj):
				for imp := range r.imports() {
					importer.Add(pass.Fset, stack[0].(*ast.File), imp.alias, imp.path)
				}
			case parsedFunc.matchesFuncReceiver(obj):
			default:
				return true
			}

			var replacement string
			replacement, err = r.print(pass.Fset, callExpr)
			if err != nil {
				return false
			}

			err = analyzeutil.ReplaceNode(pass, callExpr, replacement)
			if err != nil {
				return false
			}

			return true
		},
	)

	return err
}
