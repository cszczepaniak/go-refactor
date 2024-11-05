package replace

import "github.com/cszczepaniak/go-refactor/internal/analyzers/replace/bar"

type ReplaceInPackage struct{}

func aFunction(a, b bar.Something) {}

type aType struct {
	a bar.Something
	b int
}

func (aType) aMethod(c bar.Something) (abc bar.Something) {
	// TODO support composite literals and type casts and var decls and such
	var x bar.Something
	y := bar.Something(x)
	_ = y

	return bar.Something{}
}

type anInterface interface {
	aMethod(a, b bar.Something) (bar.Something, int)
}
