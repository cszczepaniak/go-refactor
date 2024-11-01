package replace

import (
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestReplace_Basic(t *testing.T) {
	a, err := New(Config{
		Package:     "test.com/module/basic",
		Function:    "ReplaceMe",
		Replacement: "ExampleReplacement",
	})
	require.NoError(t, err)

	analysistest.RunWithSuggestedFixes(t, analysistest.TestData(), a, "./basic")
}

func TestReplace_Args(t *testing.T) {
	a, err := New(Config{
		Package:     "test.com/module/args",
		Function:    "ReplaceMe",
		Replacement: "Replaced($arg1, $arg0)",
	})
	require.NoError(t, err)

	analysistest.RunWithSuggestedFixes(t, analysistest.TestData(), a, "./args")
}
