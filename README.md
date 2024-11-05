# go-refactor
`go-refactor` is a tool used for refactoring Go code. Its aim is to automate common refactoring
tasks that are a little more involved than just a simple find/replace.

## Installation
Currently, the best way to install `go-refactor` is to clone the repository and run `make install`.

# Supported Refactorings

## `replace`
`replace` is used to replace a function call with some transformation of it. See example usages
below.

```shell
# The arguments are available as metavariables.
go-refactor replace --func github.com/cszczepaniak/go-refactor/internal/analyzers/replace.New
--replacement 'NewWithAnotherArg("another argument", $arg0)' ./...

# You can also specify that a new import is needed. This will add
# "github.com/cszczepaniak/another/pkg" as an import and expand to "pkgname."
go-refactor replace --func github.com/cszczepaniak/go-refactor/internal/analyzers/replace.New
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


## `move`
`move` is used to move a type (and all of its methods, and optionally constructors) to a different
package and updates all references elsewhere to refer to it by its new name.
...it's not currently implemented.
