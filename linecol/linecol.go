package linecol

import (
	"unicode/utf8"
)

// LineColer calculates line and col.
type LineColer struct {
	src     string
	linePos []int
}

// NewLineColer creates LineColer.
func NewLineColer(src string) LineColer {
	linePos := []int{-1}
	pos := 0
	for pos < len(src) {
		remaining := src[pos:]
		r, size := utf8.DecodeRuneInString(remaining)
		if r == '\n' {
			linePos = append(linePos, pos)
		}
		pos += size
	}
	return LineColer{
		src:     src,
		linePos: linePos,
	}
}

// LineCol returns line and col for the given offset.
func (p *LineColer) LineCol(offset int) (line, col int) {
	lineIndex := p.findLineIndex(offset)
	if lineIndex < 0 {
		return 0, 0
	}
	line = lineIndex + 1
	lineOffset := p.linePos[lineIndex]
	col = offset - lineOffset
	return
}

func (p *LineColer) findLineIndex(offset int) int {
	if offset < 0 || offset >= len(p.src) {
		return -1
	}

	lastLineOffset := p.linePos[len(p.linePos)-1]
	if offset >= lastLineOffset {
		return len(p.linePos) - 1
	}

	low := 0
	high := len(p.linePos)
	for {
		needle := (high-low)/2 + low
		prev := needle - 1
		if prev < 0 {
			return 0
		}
		if p.linePos[prev] == offset {
			return prev
		}
		if p.linePos[needle] == offset {
			return needle
		}
		if p.linePos[prev] < offset && offset < p.linePos[needle] {
			return prev
		}
		if offset > p.linePos[needle] {
			low = needle
		} else {
			high = needle + 1
		}
	}
}
