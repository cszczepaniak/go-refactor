package replace

import (
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"strconv"
	"strings"
	"unicode"
)

func parseReplacement(replacementStr string) (parsedReplacement, error) {
	rest := replacementStr

	var pr parsedReplacement

	for len(rest) > 0 {
		if rest[0] == '$' {
			// meta vars start with $
			rest = rest[1:]

			// for now we only support argNNN
			if len(rest) < 4 || rest[:3] != "arg" {
				return parsedReplacement{}, errors.New("malformed placeholder; expected $argNNN")
			}

			rest = rest[3:]

			end := 0
			for end < len(rest) && unicode.IsDigit(rune(rest[end])) {
				end++
			}

			idx, err := strconv.Atoi(rest[:end])
			if err != nil {
				return parsedReplacement{}, err
			}

			pr.replacers = append(pr.replacers, argReplacer{index: idx})
			rest = rest[end:]
			continue
		}

		end := 0
		for end < len(rest) && rest[end] != '$' {
			end++
		}

		pr.replacers = append(pr.replacers, constantReplacer(rest[:end]))
		rest = rest[end:]
	}

	return pr, nil
}

type parsedReplacement struct {
	replacers []replacer
}

func (pr parsedReplacement) print(fset *token.FileSet, call *ast.CallExpr) (string, error) {
	sb := &strings.Builder{}
	for _, r := range pr.replacers {
		s, err := r.print(fset, call)
		if err != nil {
			return "", err
		}

		sb.WriteString(s)
	}

	return sb.String(), nil
}

type replacer interface {
	print(*token.FileSet, *ast.CallExpr) (string, error)
}

type constantReplacer string

func (cr constantReplacer) print(*token.FileSet, *ast.CallExpr) (string, error) {
	return string(cr), nil
}

type argReplacer struct {
	index int
}

func (ar argReplacer) print(fset *token.FileSet, call *ast.CallExpr) (string, error) {
	if ar.index < 0 {
		return "", errors.New("index must be greater than or equal to 0")
	}

	if ar.index >= len(call.Args) {
		return "", fmt.Errorf("index was %d but there are only %d arguments", ar.index, len(call.Args))
	}

	dst := &strings.Builder{}
	err := format.Node(dst, fset, call.Args[ar.index])
	if err != nil {
		return "", err
	}

	return dst.String(), nil
}
