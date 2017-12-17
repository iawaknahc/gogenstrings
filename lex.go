package main

import (
	"errors"
	"fmt"
	"unicode/utf8"
)

const EOF = -1

type ItemType int

const (
	ItemError ItemType = iota
	ItemEOF
	ItemComment
	ItemSpaces
	ItemString
	ItemEqualSign
	ItemSemicolon
	ItemAtSign
	ItemColon
	ItemComma
	ItemIdentifier
	ItemParenLeft
	ItemParenRight
)

type Item struct {
	Type      ItemType
	Value     string
	Err       error
	Start     int
	End       int
	StartLine int
	StartCol  int
	EndLine   int
	EndCol    int
}

func (i Item) String() string {
	switch i.Type {
	case ItemError:
		return fmt.Sprintf("%v at %v:%v", i.Err, i.EndLine, i.EndCol)
	case ItemEOF:
		return fmt.Sprintf("EOF")
	case ItemEqualSign:
		fallthrough
	case ItemSemicolon:
		return fmt.Sprintf("%v at %v:%v", i.Value, i.EndLine, i.EndCol)
	default:
		return fmt.Sprintf("%q from %v:%v to %v:%v", i.Value, i.StartLine, i.StartCol, i.EndLine, i.EndCol)
	}
}

type StateFn func(*Lexer) StateFn

type Lexer struct {
	state   StateFn
	input   string
	start   int
	pos     int
	line    int
	linePos []int
	width   int
	items   chan Item
}

func NewLexer(input string, state StateFn) Lexer {
	l := Lexer{
		state:   state,
		input:   input,
		linePos: []int{-1},
		items:   make(chan Item, 2),
	}
	go l.run()
	return l
}

func (l *Lexer) NextItem() Item {
	item := <-l.items
	return item
}

func (l *Lexer) run() {
	for s := l.state; s != nil; s = l.state {
		l.state = l.state(l)
	}
	close(l.items)
}

func (l *Lexer) Next() rune {
	if l.pos >= len(l.input) {
		return EOF
	}
	r, w := utf8.DecodeRuneInString(l.input[l.pos:])
	if r == '\n' {
		l.line += 1
		l.linePos = append(l.linePos, l.pos)
	}
	l.pos += w
	l.width = w
	return r
}

func (l *Lexer) Backup() {
	l.pos -= l.width
	r, _ := utf8.DecodeRuneInString(l.input[l.pos:])
	if r == '\n' {
		l.line -= 1
		l.linePos = l.linePos[:len(l.linePos)-1]
	}
}

func (l *Lexer) Peek() rune {
	r := l.Next()
	l.Backup()
	return r
}

func (l *Lexer) Ignore() {
	l.start = l.pos
}

func (l *Lexer) Emit(typ ItemType) {
	item := Item{
		Type:      typ,
		Value:     l.input[l.start:l.pos],
		Start:     l.start,
		End:       l.pos,
		StartLine: l.getLine(l.start),
		StartCol:  l.getCol(l.start),
		EndLine:   l.getLine(l.pos),
		EndCol:    l.getCol(l.pos),
	}
	l.start = l.pos
	l.items <- item
}

func (l *Lexer) UnexpectedToken(r rune) StateFn {
	var err error
	if r == EOF {
		err = errors.New("unexpected EOF")
	} else {
		err = errors.New(fmt.Sprintf("unexpected token `%c`", r))
	}
	item := Item{
		Type:      ItemError,
		Err:       err,
		Start:     l.start,
		End:       l.pos,
		StartLine: l.getLine(l.start),
		StartCol:  l.getCol(l.start),
		EndLine:   l.getLine(l.pos),
		EndCol:    l.getCol(l.pos),
	}
	l.start = l.pos
	l.items <- item
	return nil
}

func (l *Lexer) EOF() StateFn {
	l.Emit(ItemEOF)
	return nil
}

func (l *Lexer) getLine(pos int) int {
	for i := len(l.linePos) - 1; i >= 0; i -= 1 {
		linePos := l.linePos[i]
		if pos >= linePos {
			return i + 1
		}
	}
	return 1
}

func (l *Lexer) getCol(pos int) int {
	for i := len(l.linePos) - 1; i >= 0; i -= 1 {
		linePos := l.linePos[i]
		if pos >= linePos {
			return pos - linePos
		}
	}
	return 1
}

func isSpace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r'
}

func isIdentifierStart(r rune) bool {
	return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r == '_'
}

func isIdentifier(r rune) bool {
	return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r == '_' || r >= '0' && r <= '9'
}
