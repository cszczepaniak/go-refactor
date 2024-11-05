package main

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/cszczepaniak/go-refactor/internal/driver/driver"
	"github.com/urfave/cli/v2"
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
			Name: "replace",
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
			Action: subcommand("replace"),
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

func subcommand(name string) func(*cli.Context) error {
	return func(cctx *cli.Context) error {
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
				flags[f.Name] = cctx.String(f.Name)
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

		out, err := do(
			name,
			flags,
			cctx.Args().Slice(),
		)
		if err != nil {
			return err
		}

		if cctx.Count("verbose") > 1 {
			fmt.Println(out.Output())
		}

		if cctx.Bool("verbose") || cctx.Bool("dry-run") {
			fmt.Printf("%d issues found and fixed\n", out.Count)
		}

		return nil
	}
}
