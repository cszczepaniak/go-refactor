package analyzeutil

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImporter(t *testing.T) {
	src := `package foo

import (
	"github.com/w/x/y"
	"github.com/w/x/y/z"
)`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(
		fset,
		`foo.go`,
		[]byte(src),
		parser.AllErrors,
	)
	require.NoError(t, err)

	// We should not add anything
	imp := &Importer{}
	name := imp.Add(fset, f, "", "github.com/w/x/y")
	assert.Empty(t, name)
	assert.Empty(t, imp.filesByName)

	// We should add the new import
	name = imp.Add(fset, f, "hmm", "github.com/new/imp")
	assert.Equal(t, "hmm", name)
	assert.NotEmpty(t, imp.filesByName)

	// Adding the same import again should no-op, but return the old name
	name = imp.Add(fset, f, "anotha", "github.com/new/imp")
	assert.Equal(t, "hmm", name)
	assert.NotEmpty(t, imp.filesByName)

	assert.Len(t, imp.filesByName[`foo.go`].mutated.Imports, 3)

	assertHasImport := func(path, name string) {
		t.Helper()

		f := imp.filesByName[`foo.go`]
		require.NotNil(t, f)

		allImps := make([]string, 0, len(f.mutated.Imports))
		for _, imp := range f.mutated.Imports {
			formatted, err := FormatNode(fset, imp)
			require.NoError(t, err)
			allImps = append(allImps, formatted)
		}

		exp := &ast.ImportSpec{
			Path: &ast.BasicLit{
				Kind:  token.STRING,
				Value: fmt.Sprintf("%q", path),
			},
		}
		if name != "" {
			exp.Name = &ast.Ident{Name: name}
		}

		expFormatted, err := FormatNode(fset, exp)
		require.NoError(t, err)
		assert.Contains(t, allImps, expFormatted)
	}

	assertHasImport("github.com/w/x/y", "")
	assertHasImport("github.com/w/x/y/z", "")
	assertHasImport("github.com/new/imp", "hmm")
}
