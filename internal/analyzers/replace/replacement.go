package replace

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"iter"
	"strconv"
	"strings"
	"unicode"

	"github.com/cszczepaniak/go-refactor/internal/analyzeutil"
)

func parseReplacement(replacementStr string) (parsedReplacement, error) {
	rest := replacementStr

	var pr parsedReplacement

	for len(rest) > 0 {
		if rest[0] == '$' {
			// meta vars start with $
			rest = rest[1:]

			var metaName string
			metaName, rest = takeWhile(rest, unicode.IsLetter)

			// for now we only support argNNN
			switch metaName {
			case "arg":
				var numStr string
				numStr, rest = takeWhile(rest, unicode.IsDigit)

				idx, err := strconv.Atoi(numStr)
				if err != nil {
					return parsedReplacement{}, err
				}

				pr.replacers = append(pr.replacers, argReplacer{index: idx})
			case "recv":
				pr.replacers = append(pr.replacers, recvReplacer{})
			case "pkg":
				var err error
				rest, err = expectRune(rest, '(')
				if err != nil {
					return parsedReplacement{}, err
				}

				var pkg string
				pkg, rest = takeWhile(rest, func(r rune) bool { return r != ',' })

				rest, err = expectRune(rest, ',')
				if err != nil {
					return parsedReplacement{}, err
				}

				var name string
				name, rest = takeWhile(rest, func(r rune) bool { return r != ',' && r != ')' })

				var alias string
				if rest[0] == ',' {
					rest, _ = expectRune(rest, ',')
					alias, rest = takeWhile(rest, func(r rune) bool { return r != ')' })
				}

				rest, err = expectRune(rest, ')')
				if err != nil {
					return parsedReplacement{}, err
				}

				pr.replacers = append(pr.replacers, packageReplacer{
					path:  pkg,
					name:  name,
					alias: alias,
				})
			default:
				return parsedReplacement{}, fmt.Errorf("malformed placeholder; expected $argNNN but got $%s", metaName)
			}

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

func takeWhile(s string, fn func(r rune) bool) (string, string) {
	end := 0
	for end < len(s) && fn(rune(s[end])) {
		end++
	}
	return s[:end], s[end:]
}

func expectRune(str string, r rune) (string, error) {
	if str == "" {
		return "", fmt.Errorf("expected %c but the string was empty", r)
	}
	if rune(str[0]) != r {
		return "", fmt.Errorf("expected %c but got %c", r, str[0])
	}

	return str[1:], nil
}

type parsedReplacement struct {
	replacers []replacer
}

func (pr parsedReplacement) imports() iter.Seq[packageReplacer] {
	return func(yield func(packageReplacer) bool) {
		for _, r := range pr.replacers {
			if r, ok := r.(packageReplacer); ok {
				if !yield(r) {
					return
				}
			}
		}
	}
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

type packageReplacer struct {
	path  string
	name  string
	alias string
}

func (pr packageReplacer) print(*token.FileSet, *ast.CallExpr) (string, error) {
	return pr.name, nil
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

	return analyzeutil.FormatNode(fset, call.Args[ar.index])
}

type recvReplacer struct{}

func (r recvReplacer) print(fset *token.FileSet, call *ast.CallExpr) (string, error) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return "", nil
	}

	return analyzeutil.FormatNode(fset, sel.X)
}
