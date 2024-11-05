package driver

import (
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

func (d Driver) Execute(subcmd string, flags map[string]string, args []string) (string, error) {
	preparedArgs := make([]string, 0, len(flags)*2+2+len(args))
	preparedArgs = append(preparedArgs, "-"+subcmd, "-fix")
	for k, v := range flags {
		preparedArgs = append(preparedArgs, "-"+subcmd+"."+k, v)
	}
	preparedArgs = append(preparedArgs, args...)

	output := &strings.Builder{}
	cmd := exec.Command(d.exePath, preparedArgs...)
	cmd.Stdout = output
	cmd.Stderr = output

	err := cmd.Start()
	if err != nil {
		return "", err
	}

	state, err := cmd.Process.Wait()
	if err != nil {
		return "", err
	}

	switch state.ExitCode() {
	case 0:
		return "", ErrNoResults
	case 3:
		// Code 3 is returned when diagnostics were reported, this is our success case (when we had
		// something to fix).
		return output.String(), nil
	default:
		return "", fmt.Errorf("error running driver: %s", output)
	}
}

func (d Driver) Cleanup() error {
	return os.RemoveAll(d.dir)
}
