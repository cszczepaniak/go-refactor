package replace

import (
	"go/ast"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsingReplacement_Basic(t *testing.T) {
	r, err := parseReplacement("replace")
	require.NoError(t, err)

	res, err := r.print(nil, nil)
	require.NoError(t, err)

	assert.Equal(t, "replace", res)
}

func TestParsingReplacement_Args_Basic(t *testing.T) {
	r, err := parseReplacement("$arg0")
	require.NoError(t, err)

	res, err := r.print(token.NewFileSet(), &ast.CallExpr{
		Args: []ast.Expr{
			&ast.Ident{Name: "foo"},
		},
	})
	require.NoError(t, err)

	assert.Equal(t, "foo", res)
}

func TestParsingReplacement_Args_Multiple(t *testing.T) {
	r, err := parseReplacement("$arg0, $arg1")
	require.NoError(t, err)

	res, err := r.print(token.NewFileSet(), &ast.CallExpr{
		Args: []ast.Expr{
			&ast.Ident{Name: "foo"},
			&ast.Ident{Name: "bar"},
		},
	})
	require.NoError(t, err)

	assert.Equal(t, "foo, bar", res)
}

func TestParsingReplacement_Args_MultipleComplex(t *testing.T) {
	r, err := parseReplacement("$arg0, $arg1")
	require.NoError(t, err)

	res, err := r.print(token.NewFileSet(), &ast.CallExpr{
		Args: []ast.Expr{
			&ast.BinaryExpr{
				X:  &ast.Ident{Name: "foo"},
				Op: token.MUL,
				Y: &ast.BasicLit{
					Kind:  token.STRING,
					Value: "\"blah\"",
				},
			},
			&ast.Ident{Name: "bar"},
		},
	})
	require.NoError(t, err)

	assert.Equal(t, "foo * \"blah\", bar", res)
}

func TestParsingReplacement_Realistic(t *testing.T) {
	r, err := parseReplacement("$arg2.SomeFunction($arg1, $arg0)")
	require.NoError(t, err)

	res, err := r.print(token.NewFileSet(), &ast.CallExpr{
		Args: []ast.Expr{
			&ast.BasicLit{
				Kind:  token.INT,
				Value: "123",
			},
			&ast.Ident{Name: "bar"},
			&ast.Ident{Name: "moveThis"},
		},
	})
	require.NoError(t, err)

	assert.Equal(t, "moveThis.SomeFunction(bar, 123)", res)
}
