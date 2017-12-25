package main

import (
	"io/ioutil"
	"unicode/utf8"

	"github.com/iawaknahc/gogenstrings/errors"
)

func readFile(filename string) (string, error) {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}
	if !utf8.Valid(bytes) {
		return "", errors.File(filename, "file is not UTF-8 encoded")
	}
	return string(bytes), nil
}

func writeFile(filename, content string) error {
	// Write the file directly instead of
	// Writing to the temp file followed by a rename
	// in order to avoid cross-device link
	return ioutil.WriteFile(filename, []byte(content), 0644)
}
