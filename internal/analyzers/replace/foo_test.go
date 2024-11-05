package replace

import "github.com/cszczepaniak/go-refactor/internal/analyzers/replace/bar"

type ReplaceInPackage struct{}

func aFunction(a, b bar.Something) {}

type aType struct {
	a bar.Something
	b int
}

func (aType) aMethod(c bar.Something) (abc bar.Something) {
	var x bar.Something
	y := bar.Something(x)
	_ = y

	return bar.Something{}
}

type anInterface interface {
	aMethod(a, b bar.Something) (bar.Something, int)
}

type ReplaceMe struct{}

func aFunction2(a, b ReplaceMe) {}

type aType2 struct {
	a ReplaceMe
	b int
}

func (aType2) aMethod2(c ReplaceMe) (abc ReplaceMe) {
	var x ReplaceMe
	y := ReplaceMe(x)
	_ = y

	return ReplaceMe{}
}

type anInterface2 interface {
	aMethod(a, b ReplaceMe) (ReplaceMe, int)
}
