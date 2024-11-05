package replace

import (
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"os"
	"strings"

	"github.com/cszczepaniak/go-refactor/internal/analyzeutil"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

func New(dummy string) *analysis.Analyzer {
	flagSet := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	function := flagSet.String("func", "", "The function to replace. Format is 'github.com/package/path.FunctionName'")
	replacement := flagSet.String("replacement", "", "The replacement string. Placeholders are available (like $arg0).")

	return &analysis.Analyzer{
		Name:  "replace",
		Doc:   "Replace a function call with something else.",
		Flags: *flagSet,
		Run: func(pass *analysis.Pass) (interface{}, error) {
			if *function == "" {
				return nil, errors.New("function is required")
			}

			dot := strings.LastIndex(*function, ".")
			if dot < 0 {
				return nil, fmt.Errorf("function must be a package path and a name separated by a period, e.g. github.com/a.Function (had %s)", *function)
			}

			pkg := (*function)[:dot]
			function := (*function)[dot+1:]

			r, err := parseReplacement(*replacement)
			if err != nil {
				return nil, err
			}

			importer := &analyzeutil.Importer{}
			inspector := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

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
					if obj == nil || obj.Name() != function || obj.Pkg().Path() != pkg {
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

					for imp := range r.imports() {
						importer.Add(pass.Fset, stack[0].(*ast.File), imp.alias, imp.path)
					}

					return true
				},
			)
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
