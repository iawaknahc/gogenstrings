package linecol

import (
	"testing"
)

func TestLineColOffset(t *testing.T) {
	cases := []struct {
		src    string
		offset int
		line   int
		col    int
	}{
		{"\na", 1, 2, 1},
		{"abc\nabc\n", -1, 0, 0},
		{"abc\nabc\n", 0, 1, 1},
		{"abc\nabc\n", 1, 1, 2},
		{"abc\nabc\n", 2, 1, 3},
		{"abc\nabc\n", 3, 2, 0},
		{"abc\nabc\n", 4, 2, 1},
		{"abc\nabc\n", 5, 2, 2},
		{"abc\nabc\n", 6, 2, 3},
		{"abc\nabc\n", 7, 3, 0},
		{"abc\nabc\n", 8, 0, 0},
	}
	for _, c := range cases {
		lineColer := NewLineColer(c.src)
		line, col := lineColer.LineCol(c.offset)
		if line != c.line || col != c.col {
			t.Errorf("%q %v actual=%v:%v expected=%v:%v\n", c.src, c.offset, line, col, c.line, c.col)
		}
	}
}
