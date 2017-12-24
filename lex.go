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
	Filepath  string
	Start     int
	End       int
	StartLine int
	StartCol  int
	EndLine   int
	EndCol    int
}

func (i lexItem) unexpectedTokenErr() errFileLineCol {
	return makeErrFileLineCol(
		i.Filepath,
		i.EndLine,
		i.EndCol-1,
		fmt.Sprintf("unexpected token `%v`", i.Value),
	)
}

type stateFn func(*lexer) stateFn

type lexer struct {
	state     stateFn
	filepath  string
	input     string
	start     int
	pos       int
	width     int
	lineColer LineColer
	items     chan lexItem
}

func newLexer(input, filepath string, state stateFn) lexer {
	l := lexer{
		state:     state,
		filepath:  filepath,
		lineColer: NewLineColer(input),
		input:     input,
		items:     make(chan lexItem, 2),
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
	l.pos += w
	l.width = w
	return r
}

func (l *lexer) backup() {
	l.pos -= l.width
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
	startLine, startCol := l.lineCol(l.start)
	endLine, endCol := l.lineCol(l.pos)
	item := lexItem{
		Type:      typ,
		Value:     l.input[l.start:l.pos],
		Filepath:  l.filepath,
		Start:     l.start,
		End:       l.pos,
		StartLine: startLine,
		StartCol:  startCol,
		EndLine:   endLine,
		EndCol:    endCol,
	}
	l.start = l.pos
	l.items <- item
}

func (l *lexer) unexpectedToken(r rune) stateFn {
	startLine, startCol := l.lineCol(l.start)
	endLine, endCol := l.lineCol(l.pos)
	var err error
	if r == eof {
		err = makeErrFileLineCol(l.filepath, endLine, endCol-1, "unexpected EOF")
	} else {
		err = makeErrFileLineCol(
			l.filepath,
			endLine,
			endCol-1,
			fmt.Sprintf("unexpected token `%c`", r),
		)
	}
	item := lexItem{
		Type:      itemError,
		Err:       err,
		Start:     l.start,
		End:       l.pos,
		StartLine: startLine,
		StartCol:  startCol,
		EndLine:   endLine,
		EndCol:    endCol,
	}
	l.start = l.pos
	l.items <- item
	return nil
}

func (l *lexer) eof() stateFn {
	l.emit(itemEOF)
	return nil
}

func (l *lexer) lineCol(offset int) (line, col int) {
	return l.lineColer.LineCol(offset)
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
