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

		flags := make(map[string]string, len(cctx.Command.Flags))
		for _, f := range cctx.Command.Flags {
			switch f := f.(type) {
			case *cli.StringFlag:
				flags[f.Name] = f.Value
			case *cli.IntFlag:
				flags[f.Name] = strconv.Itoa(f.Value)
			case *cli.BoolFlag:
				flags[f.Name] = strconv.FormatBool(f.Value)
			default:
				return fmt.Errorf("unsupported flag type: %T", f)
			}
		}

		out, err := d.Execute(
			name,
			flags,
			cctx.Args().Slice(),
		)
		if err != nil {
			return err
		}

		if cctx.Bool("verbose") {
			fmt.Printf("%d issues found and fixed\n", out.Count)
		}

		return nil
	}
}
