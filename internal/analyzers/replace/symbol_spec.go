package replace

import (
	"errors"
	"go/types"
	"strings"
)

type SymbolSpec struct {
	Pkg  string
	name string

	// recv is only set for function specs
	recv string
}

func (s SymbolSpec) matchesTopLevelSymbol(obj types.Object) bool {
	return obj != nil && obj.Name() == s.name && obj.Pkg() != nil && obj.Pkg().Path() == s.Pkg
}

func (s SymbolSpec) matchesFuncReceiver(obj types.Object) bool {
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

	return recvTyp.Obj().Pkg().Path() == s.Pkg && recvTyp.Obj().Name() == s.recv
}

func ParseSymbolSpec(input string) (SymbolSpec, error) {
	dot := strings.LastIndex(input, ".")
	if dot == -1 {
		return SymbolSpec{}, errors.New("spec must be of form <package path>.<receiver (optional)>.<name>")
	}

	rest, name := input[:dot], input[dot+1:]
	slash := strings.LastIndex(rest, "/")
	dot = strings.LastIndex(rest, ".")

	if slash > dot {
		// The only dots are in the package path, there is no receiver name.
		return SymbolSpec{
			Pkg:  rest,
			name: name,
		}, nil
	}

	pkg, recv := rest[:dot], rest[dot+1:]
	return SymbolSpec{
		Pkg:  pkg,
		recv: recv,
		name: name,
	}, nil
}
