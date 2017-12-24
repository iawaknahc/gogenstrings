package main

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const eof = -1

type itemType int

const (
	itemError itemType = iota
	itemEOF
	itemComment
	itemSpaces
	itemString
	itemEqualSign
	itemSemicolon
	itemAtSign
	itemColon
	itemComma
	itemIdentifier
	itemParenLeft
	itemParenRight
)

type lexItem struct {
	Type      itemType
	Value     string
	Err       error
	Start     int
	End       int
	StartLine int
	StartCol  int
	EndLine   int
	EndCol    int
}

func (i lexItem) String() string {
	switch i.Type {
	case itemError:
		return fmt.Sprintf("%v at %v:%v", i.Err, i.EndLine, i.EndCol)
	case itemEOF:
		return fmt.Sprintf("EOF")
	case itemEqualSign:
		fallthrough
	case itemSemicolon:
		return fmt.Sprintf("%v at %v:%v", i.Value, i.EndLine, i.EndCol)
	default:
		return fmt.Sprintf("%q from %v:%v to %v:%v", i.Value, i.StartLine, i.StartCol, i.EndLine, i.EndCol)
	}
}

type stateFn func(*lexer) stateFn

type lexer struct {
	state   stateFn
	input   string
	start   int
	pos     int
	line    int
	linePos []int
	width   int
	items   chan lexItem
}

func newLexer(input string, state stateFn) lexer {
	l := lexer{
		state:   state,
		input:   input,
		linePos: []int{-1},
		items:   make(chan lexItem, 2),
	}
	go l.run()
	return l
}

func (l *lexer) nextItem() lexItem {
	item := <-l.items
	return item
}

func (l *lexer) run() {
	for s := l.state; s != nil; s = l.state {
		l.state = l.state(l)
	}
	close(l.items)
}

func (l *lexer) next() rune {
	if l.pos >= len(l.input) {
		return eof
	}
	r, w := utf8.DecodeRuneInString(l.input[l.pos:])
	if r == '\n' {
		l.line++
		l.linePos = append(l.linePos, l.pos)
	}
	l.pos += w
	l.width = w
	return r
}

func (l *lexer) backup() {
	l.pos -= l.width
	r, _ := utf8.DecodeRuneInString(l.input[l.pos:])
	if r == '\n' {
		l.line--
		l.linePos = l.linePos[:len(l.linePos)-1]
	}
}

func (l *lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

func (l *lexer) ignore() {
	l.start = l.pos
}

func (l *lexer) emit(typ itemType) {
	item := lexItem{
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

func (l *lexer) unexpectedToken(r rune) stateFn {
	var err error
	if r == eof {
		err = fmt.Errorf("unexpected EOF")
	} else {
		err = fmt.Errorf("unexpected token `%c`", r)
	}
	item := lexItem{
		Type:      itemError,
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

func (l *lexer) eof() stateFn {
	l.emit(itemEOF)
	return nil
}

func (l *lexer) getLine(pos int) int {
	for i := len(l.linePos) - 1; i >= 0; i-- {
		linePos := l.linePos[i]
		if pos >= linePos {
			return i + 1
		}
	}
	return 1
}

func (l *lexer) getCol(pos int) int {
	for i := len(l.linePos) - 1; i >= 0; i-- {
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

func lexComment(state stateFn) stateFn {
	return func(l *lexer) stateFn {
		l.next()
		l.next()
		for {
			if strings.HasPrefix(l.input[l.pos:], "*/") {
				l.next()
				l.next()
				l.emit(itemComment)
				return state
			}
			if r := l.next(); r == eof {
				return l.unexpectedToken(r)
			}
		}
	}
}

func lexSpaces(state stateFn) stateFn {
	return func(l *lexer) stateFn {
		for {
			r := l.next()
			if !isSpace(r) {
				if r != eof {
					l.backup()
				}
				if l.start < l.pos {
					l.emit(itemSpaces)
				}
				return state
			}
		}
	}
}

func lexString(state stateFn) stateFn {
	return func(l *lexer) stateFn {
		l.next()
		escaping := false
		for {
			r := l.next()
			switch r {
			case eof:
				return l.unexpectedToken(r)
			case '\\':
				escaping = !escaping
			case '"':
				if !escaping {
					l.emit(itemString)
					return state
				}
				escaping = false
			default:
				escaping = false
			}
		}
	}
}

func lexEntry(l *lexer) stateFn {
	for {
		if strings.HasPrefix(l.input[l.pos:], "/*") {
			return lexComment(lexEntry)
		}
		r := l.next()
		switch r {
		case eof:
			return l.eof()
		case '"':
			l.backup()
			return lexString(lexEntry)
		case ';':
			l.emit(itemSemicolon)
		case '=':
			l.emit(itemEqualSign)
		default:
			if isSpace(r) {
				l.backup()
				return lexSpaces(lexEntry)
			}
			return l.unexpectedToken(r)
		}
	}
}

func lexIdentifier(state stateFn) stateFn {
	return func(l *lexer) stateFn {
		for {
			r := l.next()
			if !isIdentifier(r) {
				if r != eof {
					l.backup()
				}
				l.emit(itemIdentifier)
				return state
			}
		}
	}
}

func lexRoutineCall(l *lexer) stateFn {
	for {
		r := l.next()
		switch r {
		case eof:
			return l.eof()
		case '"':
			l.backup()
			return lexString(lexRoutineCall)
		case '@':
			l.emit(itemAtSign)
		case '(':
			l.emit(itemParenLeft)
		case ')':
			l.emit(itemParenRight)
		case ':':
			l.emit(itemColon)
		case ',':
			l.emit(itemComma)
		default:
			if isSpace(r) {
				l.backup()
				return lexSpaces(lexRoutineCall)
			} else if isIdentifierStart(r) {
				return lexIdentifier(lexRoutineCall)
			} else {
				l.ignore()
			}
		}
	}
}
