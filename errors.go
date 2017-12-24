package main

import (
	"fmt"
)

type errFileLineCol struct {
	filepath string
	line     int
	col      int
	message  string
}

func (e errFileLineCol) Error() string {
	return fmt.Sprintf("%v:%v:%v: %v", e.filepath, e.line, e.col, e.message)
}

func makeErrFileLineCol(filepath string, line, col int, message string) errFileLineCol {
	return errFileLineCol{
		filepath: filepath,
		line:     line,
		col:      col,
		message:  message,
	}
}

type errFile struct {
	filepath string
	message  string
}

func (e errFile) Error() string {
	return fmt.Sprintf("%v: %v", e.filepath, e.message)
}

func makeErrFile(filepath, message string) errFile {
	return errFile{
		filepath: filepath,
		message:  message,
	}
}
