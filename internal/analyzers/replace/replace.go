package replace

import (
	"cmp"
	"errors"
	"flag"
	"fmt"
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
	replacement string

	typeName               string
	typeReplacementPackage string
	aliasTypeReplacement   bool
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

func (f flags) parseSymbolSpec() (symbolSpec, error) {
	spec, err := parseSymbolSpec(cmp.Or(f.function, f.typeName))
	if err != nil {
		return symbolSpec{}, err
	}

	if spec.recv != "" && f.typeName != "" {
		return symbolSpec{}, errors.New("receiver name not supported for type replacements")
	}

	return spec, nil
}

func New(dummy string) *analysis.Analyzer {
	var flags flags
	flagSet := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flagSet.StringVar(&flags.function, "func", "", "The function to replace. Format is 'github.com/package/path.FunctionName'")
	flagSet.StringVar(&flags.typeName, "type", "", "The type to replace. Format is 'github.com/package/path.TypeName'")
	flagSet.StringVar(&flags.typeReplacementPackage, "package", "", "The name of the package containing the replacement type. This is the alias that will be added to the import if --alias-type-replacement is set.")
	flagSet.BoolVar(&flags.aliasTypeReplacement, "alias-type-replacement", false, "Whether or not to add the import for the replacement type with an alias.")
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

			importer := &analyzeutil.Importer{}
			inspector := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

			spec, err := flags.parseSymbolSpec()
			if err != nil {
				return nil, err
			}

			switch {
			case flags.function != "":
				r, err := parseReplacement(flags.replacement)
				if err != nil {
					return nil, err
				}

				err = doFunctionReplacement(pass, spec, inspector, importer, r)
			case flags.typeName != "":
				replacement, err := parseSymbolSpec(flags.replacement)
				if err != nil {
					return nil, fmt.Errorf("error parsing type replacement (must be a package + symbol): %w", err)
				}
				alias := flags.typeReplacementPackage
				if !flags.aliasTypeReplacement {
					alias = ""
				}
				err = doTypeReplacement(pass, spec, replacement, alias, inspector, importer)
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
	parsedFunc symbolSpec,
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

func doTypeReplacement(
	pass *analysis.Pass,
	spec symbolSpec,
	replacement symbolSpec,
	replacementPackageName string,
	inspector *inspector.Inspector,
	importer *analyzeutil.Importer,
) error {
	var err error
	inspector.WithStack(
		[]ast.Node{&ast.Field{}},
		func(n ast.Node, push bool, stack []ast.Node) bool {
			if !push {
				return false
			}

			field := n.(*ast.Field)

			sel, ok := field.Type.(*ast.SelectorExpr)
			if !ok {
				return true
			}

			obj := pass.TypesInfo.ObjectOf(sel.Sel)

			switch {
			case spec.matchesTopLevelSymbol(obj):
				importer.Add(pass.Fset, stack[0].(*ast.File), replacementPackageName, replacement.pkg)
				err = analyzeutil.ReplaceNode(pass, field.Type, replacementPackageName+"."+replacement.name)

				// Whether or not there's an error, there's no need to descend further into a Field.
				return false
			default:
				return true
			}
		},
	)

	return err
}

type symbolSpec struct {
	pkg  string
	name string

	// recv is only set for function specs
	recv string
}

func (s symbolSpec) matchesTopLevelSymbol(obj types.Object) bool {
	return obj != nil && obj.Name() == s.name && obj.Pkg() != nil && obj.Pkg().Path() == s.pkg
}

func (s symbolSpec) matchesFuncReceiver(obj types.Object) bool {
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

func parseSymbolSpec(input string) (symbolSpec, error) {
	dot := strings.LastIndex(input, ".")
	if dot == -1 {
		return symbolSpec{}, errors.New("spec must be of form <package path>.<receiver (optional)>.<name>")
	}

	rest, name := input[:dot], input[dot+1:]
	slash := strings.LastIndex(rest, "/")
	dot = strings.LastIndex(rest, ".")

	if slash > dot {
		// The only dots are in the package path, there is no receiver name.
		return symbolSpec{
			pkg:  rest,
			name: name,
		}, nil
	}

	pkg, recv := rest[:dot], rest[dot+1:]
	return symbolSpec{
		pkg:  pkg,
		recv: recv,
		name: name,
	}, nil
}
