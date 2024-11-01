package main

import (
	"github.com/cszczepaniak/go-refactor/internal/analyzers/replace"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(replace.New())
}
