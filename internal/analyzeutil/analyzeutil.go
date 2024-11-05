package analyzeutil

import (
	"go/ast"
	"go/format"
	"go/token"
	"strings"

	"golang.org/x/tools/go/analysis"
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

func ReplaceNode(pass *analysis.Pass, n ast.Node, replaceWith string) error {
	curr, err := FormatNode(pass.Fset, n)
	if err != nil {
		return err
	}

	msg := curr + " => " + replaceWith

	pass.Report(
		analysis.Diagnostic{
			Pos:     n.Pos(),
			End:     n.End(),
			Message: msg,
			SuggestedFixes: []analysis.SuggestedFix{{
				Message: msg,
				TextEdits: []analysis.TextEdit{{
					Pos:     n.Pos(),
					End:     n.End(),
					NewText: []byte(replaceWith),
				}},
			}},
		},
	)

	return nil
}
