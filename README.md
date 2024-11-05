# go-refactor
`go-refactor` is a tool used for refactoring Go code. Its aim is to automate common refactoring
tasks that are a little more involved than just a simple find/replace.

## Installation
Currently, the best way to install `go-refactor` is to clone the repository and run `make install`.

# Supported Refactorings

## `replacecall`
`replacecall` is used to replace function calls with some transformation of those calls. See example usages
below.

```shell
# The arguments are available as metavariables.
go-refactor replacecall \
    --func github.com/cszczepaniak/go-refactor/internal/analyzers/replace.New \
    --replacement 'NewWithAnotherArg("another argument", $arg0)' ./...

# You can also specify that a new import is needed. This will add
# "github.com/cszczepaniak/another/pkg" as an import and expand to "pkgname."
go-refactor replacecall \
    --func github.com/cszczepaniak/go-refactor/internal/analyzers/replace.New \
    --replacement '$pkg(github.com/cszczepaniak/another/pkg,pkgname).New($arg0)' ./...
```

There are metavariables available within the replacement string. The table below enumerates them.

| Meta Variable | Value |
| - | - |
| `$arg<n>` | The nth input argument of the function (0-indexed). Examples: `$arg0`, `$arg11` |
| `$recv` | The receiver of the function call. If the call has no receiver, it's an empty string. |
| `$recvdot` | Same as `$recv`, but followed by `.` if the receiver is present. This is useful for replacing top-level functions that may be imported under different aliases in different packages and/or replacing calls to functions in their own package. For example, when replacing a function called `Example` in a package called `mypackage`, `$recvdotNewExample` will expand to `NewExample` within `mypackage`, `mypackage.NewExample` in a package that imports `mypackage` with no alias, and `mypackage2.NewExample` in a package that imports `mypackage` with an alias of `mypackage2`. |
| `$pkg(path,name)` | A symbol from another package. An import will be added for the package if needed. |
| `$pkg(path,name,alias)` | A symbol from another package. An import with the given alias will be added for the package if needed. |


## `replacetype`
`replacetype` is used to replace references to a type with references to another type. This includes
struct fields, function input/output arguments, var declarations, type casts, etc. See example
usages below.

```shell
# A simple replacement of one type with another
go-refactor replacetype \
    --type github.com/cszczepaniak/go-refactor/internal/analyzers/replace.TypeA \
    --replacement github.com/cszczepaniak/go-refactor/internal/analyzers/replace.TypeB ./...

# Optionally specify an import alias to use when importing the package with the new type. Note that
# if a particular file already has an import for the new type's package, the alias will not be used.
go-refactor replacetype \
    --type github.com/cszczepaniak/go-refactor/internal/analyzers/replace.TypeA \
    --replacement github.com/cszczepaniak/go-refactor/internal/analyzers/anotherpkg.TypeB \
    --import-alias aliasme ./...
```
