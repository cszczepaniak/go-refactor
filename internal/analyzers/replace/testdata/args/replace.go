package basic

import "fmt"

func ReplaceMe(a string, b bool) bool {
	return false
}

func foobar() {
	r := ReplaceMe(fmt.Sprint("abc"), true != false) // want "replace the function call"
	_ = r
}
