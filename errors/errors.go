package errors

import (
	"fmt"
)

// ErrFileLineCol tells which file, which line and which column has an error.
type ErrFileLineCol struct {
	filepath string
	line     int
	col      int
	message  string
}

func (e ErrFileLineCol) Error() string {
	return fmt.Sprintf("%v:%v:%v: %v", e.filepath, e.line, e.col, e.message)
}

// FileLineCol creates ErrFileLineCol.
func FileLineCol(filepath string, line, col int, message string) ErrFileLineCol {
	return ErrFileLineCol{
		filepath: filepath,
		line:     line,
		col:      col,
		message:  message,
	}
}

// ErrFile tells which file has an error.
type ErrFile struct {
	filepath string
	message  string
}

func (e ErrFile) Error() string {
	return fmt.Sprintf("%v: %v", e.filepath, e.message)
}

// File creates ErrFile.
func File(filepath, message string) ErrFile {
	return ErrFile{
		filepath: filepath,
		message:  message,
	}
}
