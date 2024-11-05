package analyzeutil

import (
	"go/ast"
	"go/format"
	"go/token"
	"strings"
)

func FormatNode(fset *token.FileSet, n ast.Node) (string, error) {
	dst := &strings.Builder{}
	err := format.Node(dst, fset, n)
	if err != nil {
		return "", err
	}

	return dst.String(), nil
}

func PrintReplacement(fset *token.FileSet, n ast.Node, replaceWith string) (string, error) {
	curr, err := FormatNode(fset, n)
	if err != nil {
		return "", err
	}

	return curr + " => " + replaceWith, nil
}
