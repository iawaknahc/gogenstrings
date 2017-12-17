package main

import (
	"fmt"
	"io/ioutil"
	"os"
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

func WriteFile(filename, content string) (err error) {
	tempfile, err := ioutil.TempFile("", "genstrings")
	if err != nil {
		return err
	}
	tempfilePath := tempfile.Name()
	_, err = tempfile.WriteString(content)
	if err != nil {
		return err
	}
	err = tempfile.Close()
	if err != nil {
		return err
	}
	err = os.Rename(tempfilePath, filename)
	return err
}
