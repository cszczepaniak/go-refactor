package driver

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

//go:embed bin/driver
var driverExe []byte

type Driver struct {
	dir     string
	exePath string
}

func Setup() (Driver, error) {
	// Because of the limited flexibility of analysis/multichecker, we're going to embed it in this
	// executable and run it from a subprocess. This allows us to better manipulate
	// stdout/stderr/the exit code. The cost is that building this program is a lot more complicated
	// and precludes building via go install; instead, users will have to clone this repo and run
	// the custom build target in the Makefile.
	td, err := os.MkdirTemp("", "")
	if err != nil {
		return Driver{}, err
	}

	d := Driver{
		dir:     td,
		exePath: filepath.Join(td, "go_refactoring_driver"),
	}

	// TODO ModePerm is very permissive
	err = os.WriteFile(d.exePath, driverExe, os.ModePerm)
	if err != nil {
		return Driver{}, err
	}

	return d, nil
}

var (
	ErrNoResults = errors.New("no results")
)

type Result struct {
	output *strings.Builder
	Count  int
}

func (r *Result) Output() string {
	if r.output == nil {
		return ""
	}
	return r.output.String()
}

func (r *Result) Write(b []byte) (int, error) {
	if r.output == nil {
		r.output = &strings.Builder{}
	}
	r.Count += bytes.Count(b, []byte{'\n'})
	return r.output.Write(b)
}

func (d Driver) Execute(subcmd string, flags map[string]string, args []string) (*Result, error) {
	return d.execute(subcmd, flags, args, false)

}

func (d Driver) Preview(subcmd string, flags map[string]string, args []string) (*Result, error) {
	return d.execute(subcmd, flags, args, true)
}

func (d Driver) execute(subcmd string, flags map[string]string, args []string, dryrun bool) (*Result, error) {
	if len(args) == 0 {
		return nil, errors.New("must provide at least one argument specifying a package path to run the tool over")
	}

	preparedArgs := make([]string, 0, len(flags)+2+len(args))
	preparedArgs = append(preparedArgs, "-"+subcmd)
	if !dryrun {
		preparedArgs = append(preparedArgs, "-fix")
	}
	for k, v := range flags {
		preparedArgs = append(preparedArgs, "-"+subcmd+"."+k+"="+v)
	}
	preparedArgs = append(preparedArgs, args...)

	output := &Result{}
	cmd := exec.Command(d.exePath, preparedArgs...)
	cmd.Stdout = output
	cmd.Stderr = output

	err := cmd.Start()
	if err != nil {
		return nil, err
	}

	state, err := cmd.Process.Wait()
	if err != nil {
		return nil, err
	}

	switch state.ExitCode() {
	case 0:
		return nil, fmt.Errorf("execute: %w\n%s", ErrNoResults, output.Output())
	case 3:
		// Code 3 is returned when diagnostics were reported, this is our success case (when we had
		// something to fix).
		return output, nil
	default:
		return nil, fmt.Errorf("error running driver: %s", output.Output())
	}
}

func (d Driver) Cleanup() error {
	return os.RemoveAll(d.dir)
}
