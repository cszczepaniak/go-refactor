package main

import (
	"github.com/cszczepaniak/go-refactor/internal/analyzers/replace"
	"golang.org/x/tools/go/analysis/multichecker"
)

func main() {
	multichecker.Main(
		replace.New(""),
	)
}
