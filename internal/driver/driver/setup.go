package driver

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

func (d Driver) Execute(subcmd string, args map[string]string) (string, error) {
	preparedArgs := make([]string, 0, len(args)*2+2)
	preparedArgs = append(preparedArgs, "-"+subcmd, "-fix")
	for k, v := range args {
		preparedArgs = append(preparedArgs, "-"+subcmd+"."+k, v)
	}

	out, err := exec.Command(d.exePath, preparedArgs...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error running driver: %w\n%s", err, string(out))
	}

	return string(out), nil
}

func (d Driver) Cleanup() error {
	return os.RemoveAll(d.dir)
}
