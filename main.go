package main

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"maps"
	"os"
	"strconv"

	"github.com/cszczepaniak/go-refactor/internal/analyzers/replace"
	"github.com/cszczepaniak/go-refactor/internal/driver/driver"
	"github.com/urfave/cli/v2"
	"golang.org/x/tools/go/packages"
)

func main() {
	app := &cli.App{
		Name: "go-refactor",
		Args: true,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "verbose",
				Aliases: []string{"v"},
			},
			&cli.BoolFlag{
				Name:    "dry-run",
				Aliases: []string{"d"},
			},
		},
		Commands: []*cli.Command{{
			Name: "replacecall",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "func",
					Required: true,
				},
				&cli.StringFlag{
					Name:     "replacement",
					Required: true,
				},
			},
			Action: func(cctx *cli.Context) error {
				return runSubcommand(cctx, "replacecall", nil)
			},
		}, {
			Name: "replacetype",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "type",
					Required: true,
				},
				&cli.StringFlag{
					Name:     "import-alias",
					Required: false,
					Usage:    "If replacing a type, specifies whether or not the value for --package should be added as an alias to the import",
				},
				&cli.StringFlag{
					Name:     "replacement",
					Required: true,
				},
			},
			Action: func(cctx *cli.Context) error {
				spec, err := replace.ParseSymbolSpec(cctx.String("replacement"))
				if err != nil {
					return err
				}

				pkgName, err := loadPackageName(spec.Pkg)
				if err != nil {
					return err
				}

				return runSubcommand(cctx, "replacetype", map[string]string{
					"replacement-package-name": pkgName,
				})
			},
		}},
		Before: func(c *cli.Context) error {
			d, err := driver.Setup()
			if err != nil {
				return err
			}

			c.Context = context.WithValue(c.Context, "driver", d)
			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func runSubcommand(cctx *cli.Context, name string, extraFlags map[string]string) error {
	d, ok := cctx.Context.Value("driver").(driver.Driver)
	if !ok {
		return errors.New("dev error: driver not found")
	}

	globalFlagNames := make(map[string]struct{}, len(cctx.App.Flags))
	for _, f := range cctx.App.Flags {
		for _, n := range f.Names() {
			globalFlagNames[n] = struct{}{}
		}
	}

	flags := make(map[string]string, len(cctx.Command.Flags))
	for _, f := range cctx.Command.Flags {
		skip := false
		for _, n := range f.Names() {
			_, skip = globalFlagNames[n]
			if skip {
				break
			}
		}
		if skip {
			continue
		}

		switch f := f.(type) {
		case *cli.StringFlag:
			val := cctx.String(f.Name)
			if val != "" {
				flags[f.Name] = val
			}
		case *cli.IntFlag:
			flags[f.Name] = strconv.Itoa(cctx.Int(f.Name))
		case *cli.BoolFlag:
			flags[f.Name] = strconv.FormatBool(cctx.Bool(f.Name))
		default:
			return fmt.Errorf("unsupported flag type: %T", f)
		}
	}

	var do func(string, map[string]string, []string) (*driver.Result, error)
	if cctx.Bool("dry-run") {
		do = d.Preview
	} else {
		do = d.Execute
	}

	maps.Insert(flags, maps.All(extraFlags))
	out, err := do(
		name,
		flags,
		cctx.Args().Slice(),
	)
	if err != nil {
		return err
	}

	if cctx.Bool("verbose") || cctx.Bool("dry-run") {
		fmt.Println(out.Output())
		fmt.Printf("%d issues found and fixed\n", out.Count)
	}

	return nil
}

func loadPackageName(path string) (string, error) {
	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.NeedName,
	}, path)
	if err != nil {
		fmt.Println("error loading", path, err)
		return "", err
	}

	if len(pkgs) != 1 {
		return "", errors.New("loaded an unexpected number of packages")
	}

	pkg := pkgs[0]

	if len(pkgs[0].Errors) != 0 {
		errs := make([]error, 0, len(pkg.Errors))
		for _, e := range pkg.Errors {
			errs = append(errs, e)
		}

		return "", fmt.Errorf("error(s) loading packages: %w", errors.Join(errs...))
	}

	return pkgs[0].Name, nil
}
