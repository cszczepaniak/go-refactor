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

type flags struct {
	function    string
	typeName    string
	replacement string
}

func (f flags) validate() error {
	if f.replacement == "" {
		return errors.New("replacement must be provided")
	}

	if f.function == "" && f.typeName == "" {
		return errors.New("either function or type must be provided")
	}

	if f.function != "" && f.typeName != "" {
		return errors.New("either function or type must be provided, but not both")
	}

	return nil
}

func New(dummy string) *analysis.Analyzer {
	var flags flags
	flagSet := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flagSet.StringVar(&flags.function, "func", "", "The function to replace. Format is 'github.com/package/path.FunctionName'")
	flagSet.StringVar(&flags.typeName, "type", "", "The type to replace. Format is 'github.com/package/path.TypeName'")
	flagSet.StringVar(&flags.replacement, "replacement", "", "The replacement string. Placeholders are available (like $arg0).")

	return &analysis.Analyzer{
		Name:  "replace",
		Doc:   "Replace a function call with something else.",
		Flags: *flagSet,
		Run: func(pass *analysis.Pass) (interface{}, error) {
			err := flags.validate()
			if err != nil {
				return nil, err
			}

			r, err := parseReplacement(flags.replacement)
			if err != nil {
				return nil, err
			}

			importer := &analyzeutil.Importer{}
			inspector := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

			switch {
			case flags.function != "":
				err = doFunctionReplacement(pass, flags.function, inspector, importer, r)
			default:
				return nil, errors.New("dev error: unknown case")
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
	function string,
	inspector *inspector.Inspector,
	importer *analyzeutil.Importer,
	r parsedReplacement,
) error {
	parsedFunc, err := parseFunction(function)
	if err != nil {
		return err
	}

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
		return err
	}

	return nil
}

type funcSpec struct {
	pkg  string
	recv string
	name string
}

func (s funcSpec) matchesTopLevel(obj types.Object) bool {
	return obj != nil && obj.Name() == s.name && obj.Pkg() != nil && obj.Pkg().Path() == s.pkg
}

func (s funcSpec) matchesReceiver(obj types.Object) bool {
	if obj.Name() != s.name {
		return false
	}

	sig, ok := obj.(*types.Func)
	if !ok {
		return false
	}

	recv := sig.Signature().Recv()
	if recv == nil {
		return false
	}

	recvTyp, ok := recv.Type().(*types.Named)
	if !ok {
		return false
	}

	return recvTyp.Obj().Pkg().Path() == s.pkg && recvTyp.Obj().Name() == s.recv
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

	pkg, recv := rest[:dot], rest[dot+1:]
	return funcSpec{
		pkg:  pkg,
		recv: recv,
		name: name,
	}, nil
}
