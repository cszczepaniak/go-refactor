# go-refactor
`go-refactor` is a tool used for refactoring Go code. Its aim is to automate common refactoring
tasks that are a little more involved than just a simple find/replace.

## `replace`
`replace` is used to replace a function call with some transformation of it. See example usages
below.

```shell
# The arguments are available as metavariables.
go-refactor replace --func github.com/cszczepaniak/go-refactor/internal/analyzers/replace.New
--replacement 'NewWithAnotherArg("another argument", $arg0)'

# You can also specify that a new import is needed. This will add
# "github.com/cszczepaniak/another/pkg" as an import and expand to "pkgname."
go-refactor replace --func github.com/cszczepaniak/go-refactor/internal/analyzers/replace.New
--replacement '$pkg(github.com/cszczepaniak/another/pkg,pkgname).New($arg0)'
```

There are metavariables available within the replacement string. The table below enumerates them.

| Meta Variable | Value |
| - | - |
| `$argN` | The nth input argument of the function (0-indexed) |
| `$recv` | The receiver of the function call. If the call has no receiver, it's an empty string. |
| `$pkg(path,name)` | A symbol from another package. An import will be added for the package if
needed. |
| `$pkg(path,name,alias)` | A symbol from another package. An import with the given alias will be
added for the package if needed. |
