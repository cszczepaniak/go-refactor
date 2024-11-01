package basic

func ReplaceMe(a, b, c int) bool {
	return false
}

func foobar() {
	r := ReplaceMe(1, 2, 3) // want "replace the function call"
	_ = r
}
