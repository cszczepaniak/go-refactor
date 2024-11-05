package main

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"

	"github.com/cszczepaniak/go-refactor/internal/driver/driver"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name: "go-refactor",
		Args: true,
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
			Action: func(ctx *cli.Context) error {
				d, ok := ctx.Context.Value("driver").(driver.Driver)
				if !ok {
					return errors.New("dev error: driver not found")
				}

				fmt.Println("ARGS", ctx.Args().Slice())

				out, err := d.Execute(
					"replace",
					map[string]string{
						"func":        ctx.String("func"),
						"replacement": ctx.String("replacement"),
					},
					ctx.Args().Slice(),
				)
				if err != nil {
					return err
				}

				fmt.Println(out)
				return nil
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
