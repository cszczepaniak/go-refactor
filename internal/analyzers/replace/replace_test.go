package replace

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestReplace_Basic(t *testing.T) {
	a := New()
	a.Flags.Set("func", "test.com/module/basic.ReplaceMe")
	a.Flags.Set("replacement", "ExampleReplacement")

	analysistest.RunWithSuggestedFixes(t, analysistest.TestData(), a, "./basic")
}

func TestReplace_Args(t *testing.T) {
	a := New()
	a.Flags.Set("func", "test.com/module/args.ReplaceMe")
	a.Flags.Set("replacement", "Replaced($arg1, $arg0)")

	analysistest.RunWithSuggestedFixes(t, analysistest.TestData(), a, "./args")
}
