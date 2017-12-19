package main

import (
	"fmt"
	"io/ioutil"
	"unicode/utf8"
)

func ReadFile(filename string) (string, error) {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}
	if !utf8.Valid(bytes) {
		return "", fmt.Errorf("%v is not UTF-8 encoded", filename)
	}
	return string(bytes), nil
}

func WriteFile(filename, content string) error {
	// Write the file directly instead of
	// Writing to the temp file followed by a rename
	// in order to avoid cross-device link
	return ioutil.WriteFile(filename, []byte(content), 0644)
}
