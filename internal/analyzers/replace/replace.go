package replace

import (
	"errors"
	"flag"
	"go/ast"
	"go/types"
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

			parsedFunc, err := parseFunction(*function)
			if err != nil {
				return nil, err
			}

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
					if !parsedFunc.matchesTopLevel(obj) {
						return true
					}

					switch {
					case parsedFunc.matchesTopLevel(obj):
						for imp := range r.imports() {
							importer.Add(pass.Fset, stack[0].(*ast.File), imp.alias, imp.path)
						}
					case parsedFunc.matchesReceiver(obj):
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

type funcSpec struct {
	pkg  string
	recv string
	name string
}

func (s funcSpec) matchesTopLevel(obj types.Object) bool {
	return obj != nil && obj.Name() == s.name && obj.Pkg().Path() == s.pkg
}

func (s funcSpec) matchesReceiver(obj types.Object) bool {
	sig, ok := obj.(*types.Func)
	if !ok {
		return false
	}

	recv := sig.Signature().Recv()
	return recv != nil && recv.Pkg().Path() == s.pkg && recv.Name() == s.recv && obj.Name() == s.name
}

func parseFunction(input string) (funcSpec, error) {
	dot := strings.LastIndex(input, ".")
	if dot == -1 {
		return funcSpec{}, errors.New("function must be of form <package path>.<receiver (optional)>.<name>")
	}

	rest, name := input[:dot], input[dot+1:]
	slash := strings.LastIndex(rest, "/")
	dot = strings.LastIndex(rest, ".")

	if slash > dot {
		// The only dots are in the package path, there is no receiver name.
		return funcSpec{
			pkg:  rest,
			name: name,
		}, nil
	}

	pkg, recv := input[:dot], input[dot+1:]
	return funcSpec{
		pkg:  pkg,
		recv: recv,
		name: name,
	}, nil
}
