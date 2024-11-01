package replace

import (
	"flag"
	"fmt"
	"go/ast"
	"os"
	"strings"

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

			inspector := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

			inspector.Nodes(
				[]ast.Node{&ast.CallExpr{}},
				func(n ast.Node, push bool) bool {
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

					pass.Report(
						analysis.Diagnostic{
							Pos:     callExpr.Pos(),
							End:     callExpr.End(),
							Message: "replace the function call",
							SuggestedFixes: []analysis.SuggestedFix{{
								Message: "replace the function call",
								TextEdits: []analysis.TextEdit{{
									Pos:     callExpr.Pos(),
									End:     callExpr.End(),
									NewText: []byte(replacement),
								}},
							}},
						},
					)

					return true
				},
			)
			if err != nil {
				return nil, err
			}

			return nil, nil
		},
		Requires: []*analysis.Analyzer{
			inspect.Analyzer,
		},
	}
}
