package main

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

//go:embed bin/driver
var exe []byte

func main() {
	// Because of the limited flexibility of analysis/multichecker, we're going to embed it in this
	// executable and run it from a subprocess. This allows us to better manipulate
	// stdout/stderr/the exit code. The cost is that building this program is a lot more complicated
	// and precludes building via go install; instead, users will have to clone this repo and run
	// the custom build target in the Makefile.
	td, err := os.MkdirTemp("", "")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(td)

	driver := filepath.Join(td, "go_refactoring_driver")
	// TODO ModePerm is very permissive
	err = os.WriteFile(driver, exe, os.ModePerm)
	if err != nil {
		panic(err)
	}

	out, err := exec.Command(driver, "./...").CombinedOutput()
	fmt.Println(string(out))
	if err != nil {
		panic(err)
	}
	fmt.Println(string(out))
}
