package main

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/iawaknahc/gogenstrings/errors"
	"github.com/iawaknahc/gogenstrings/linecol"
)

const eof = -1

type itemType int

const (
	itemError itemType = iota
	itemEOF
	itemComment
	itemSpaces
	itemString
	itemBareString
	itemEqualSign
	itemSemicolon
	itemAtSign
	itemColon
	itemComma
	itemIdentifier
	itemParenLeft
	itemParenRight
	itemBraceLeft
	itemBraceRight
	itemLessThanSign
	itemGreaterThanSign
)

func (v itemType) String() string {
	switch v {
	case itemError:
		return "err"
	case itemEOF:
		return "EOF"
	case itemComment:
		return "<comment>"
	case itemSpaces:
		return "<space>"
	case itemString:
		return "<string>"
	case itemBareString:
		return "<bare-string>"
	case itemEqualSign:
		return "="
	case itemSemicolon:
		return ";"
	case itemAtSign:
		return "@"
	case itemColon:
		return ":"
	case itemComma:
		return ","
	case itemIdentifier:
		return "<ident>"
	case itemParenLeft:
		return "("
	case itemParenRight:
		return ")"
	case itemBraceLeft:
		return "{"
	case itemBraceRight:
		return "}"
	case itemLessThanSign:
		return "<"
	case itemGreaterThanSign:
		return ">"
	}
	return "<unknown>"
}

type lexItem struct {
	Type      itemType
	Value     string
	RawValue  string
	Err       error
	Filepath  string
	Start     int
	End       int
	StartLine int
	StartCol  int
	EndLine   int
	EndCol    int
}

func (v lexItem) String() string {
	return fmt.Sprintf("%v:%v:%v", v.StartLine, v.StartCol, v.Type)
}

func (v lexItem) unexpectedTokenErr() errors.ErrFileLineCol {
	return errors.FileLineCol(
		v.Filepath,
		v.StartLine,
		v.StartCol,
		fmt.Sprintf("unexpected token `%v`", v.Type),
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
	lineColer linecol.LineColer
	items     chan lexItem
}

func newLexer(input, filepath string, state stateFn) lexer {
	l := lexer{
		state:     state,
		filepath:  filepath,
		lineColer: linecol.NewLineColer(input),
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
	l.emitValue(typ, l.input[l.start:l.pos])
}

func (l *lexer) emitValue(typ itemType, value string) {
	startLine, startCol := l.lineCol(l.start)
	endLine, endCol := l.lineCol(l.pos)
	item := lexItem{
		Type:      typ,
		RawValue:  l.input[l.start:l.pos],
		Value:     value,
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

func (l *lexer) unterminatedStringLiteral() stateFn {
	return l.emitError("unterminated string literal", true)
}

func (l *lexer) invalidUnicodeEscape() stateFn {
	return l.emitError("invalid unicode escape", true)
}

func (l *lexer) invalidUTF16Escape() stateFn {
	return l.emitError("invalid UTF-16 escape", true)
}

func (l *lexer) invalidUniversalCharacterName() stateFn {
	return l.emitError("invalid universal character name", true)
}

func (l *lexer) invalidEscape() stateFn {
	return l.emitError("invalid escape", true)
}

func (l *lexer) emitError(msg string, atStart bool) stateFn {
	startLine, startCol := l.lineCol(l.start)
	endLine, endCol := l.lineCol(l.pos)
	var err error
	if atStart {
		err = errors.FileLineCol(l.filepath, startLine, startCol, msg)
	} else {
		err = errors.FileLineCol(l.filepath, endLine, endCol-1, msg)
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

func (l *lexer) unexpectedToken(r rune) stateFn {
	var msg string
	if r == eof {
		msg = "unexpected EOF"
	} else {
		msg = fmt.Sprintf("unexpected token `%c`", r)
	}
	return l.emitError(msg, false)
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
	return r >= 'a' && r <= 'z' ||
		r >= 'A' && r <= 'Z' ||
		r == '_'
}

func isIdentifier(r rune) bool {
	return r >= 'a' && r <= 'z' ||
		r >= 'A' && r <= 'Z' ||
		r >= '0' && r <= '9' ||
		r == '_'
}

func isASCIIPlistBareString(r rune) bool {
	// From the behavior of PLUTIL(1)
	return r >= 'a' && r <= 'z' ||
		r >= 'A' && r <= 'Z' ||
		r >= '0' && r <= '9' ||
		r == '$' || r == '-' || r == '_' || r == '.' || r == ':' || r == '/'
}

func isHex(r rune) bool {
	return r >= 'a' && r <= 'f' ||
		r >= 'A' && r <= 'F' ||
		r >= '0' && r <= '9'
}

func isOctal(r rune) bool {
	return r >= '0' && r <= '7'
}

func isValidUniversalCharacterName(r rune) bool {
	// From C99 spec
	if r == 0x0024 || r == 0x0040 || r == 0x0060 {
		return true
	}
	if r < 0x00A0 || r >= 0xD800 && r <= 0xDFFF {
		return false
	}
	return utf8.ValidRune(r)
}

func isValidEscapedRune(r rune) bool {
	return r >= 0 && r <= 127
}

func lexComment(state stateFn) stateFn {
	return func(l *lexer) stateFn {
		l.next()
		l.next()
		for {
			if strings.HasPrefix(l.input[l.pos:], "*/") {
				l.next()
				l.next()
				value := l.input[l.start+2 : l.pos-2]
				l.emitValue(itemComment, value)
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

func lexHexDigits(l *lexer, min, max int) (rune, bool) {
	hexDigits := []rune{}
	for i := 0; i < max; i++ {
		hexDigit := l.next()
		if !isHex(hexDigit) {
			l.backup()
			break
		}
		hexDigits = append(hexDigits, hexDigit)
	}
	if len(hexDigits) < min || len(hexDigits) > max {
		return 0, false
	}
	value, err := strconv.ParseInt(string(hexDigits), 16, 32)
	if err != nil {
		return 0, false
	}
	return rune(value), true
}

func lexOctalDigits(l *lexer, min, max int) (rune, bool) {
	octalDigits := []rune{}
	for i := 0; i < max; i++ {
		octalDigit := l.next()
		if !isOctal(octalDigit) {
			l.backup()
			break
		}
		octalDigits = append(octalDigits, octalDigit)
	}
	if len(octalDigits) < min || len(octalDigits) > max {
		return 0, false
	}
	value, err := strconv.ParseInt(string(octalDigits), 8, 32)
	if err != nil {
		return 0, false
	}
	return rune(value), true
}

func lexStringSwift(state stateFn) stateFn {
	// https://github.com/apple/swift/blob/master/lib/Parse/Lexer.cpp
	return func(l *lexer) stateFn {
		l.next()
		runes := []rune{}
		for {
			r := l.next()
			switch r {
			case eof, '\n', '\r':
				return l.unterminatedStringLiteral()
			case '"':
				l.emitValue(itemString, string(runes))
				return state
			case '\\':
				nextRune := l.next()
				switch nextRune {
				case eof:
					return l.unterminatedStringLiteral()
				case '\\':
					runes = append(runes, '\\')
				case '0':
					runes = append(runes, rune(0))
				case 't':
					runes = append(runes, '\t')
				case 'r':
					runes = append(runes, '\r')
				case 'n':
					runes = append(runes, '\n')
				case '\'':
					runes = append(runes, '\'')
				case '"':
					runes = append(runes, '"')
				case 'u':
					leftBrace := l.next()
					if leftBrace != '{' {
						return l.invalidUnicodeEscape()
					}
					hexDigits := []rune{}
				Loop:
					for i := 0; i < 8; i++ {
						hexDigit := l.next()
						switch hexDigit {
						case '}':
							l.backup()
							break Loop
						default:
							if !isHex(hexDigit) {
								return l.invalidUnicodeEscape()
							}
							hexDigits = append(hexDigits, hexDigit)
						}
					}
					if rightBrace := l.next(); rightBrace != '}' {
						return l.invalidUnicodeEscape()
					}
					if len(hexDigits) < 1 || len(hexDigits) > 8 {
						return l.invalidUnicodeEscape()
					}
					codePointInt64, err := strconv.ParseInt(string(hexDigits), 16, 32)
					if err != nil {
						return l.invalidUnicodeEscape()
					}
					unsafeRune := rune(codePointInt64)
					if !utf8.ValidRune(unsafeRune) {
						return l.invalidUnicodeEscape()
					}
					runes = append(runes, unsafeRune)
				default:
					return l.invalidEscape()
				}
			default:
				runes = append(runes, r)
			}
		}
	}
}

func lexStringObjc(state stateFn) stateFn {
	// Based on C99 spec
	return func(l *lexer) stateFn {
		l.next()
		runes := []rune{}
		for {
			r := l.next()
			switch r {
			case eof, '\n', '\r':
				return l.unterminatedStringLiteral()
			case '"':
				l.emitValue(itemString, string(runes))
				return state
			case '\\':
				nextRune := l.next()
				switch nextRune {
				case eof:
					return l.unterminatedStringLiteral()
				case '\\':
					runes = append(runes, '\\')
				case 'a':
					runes = append(runes, '\a')
				case 'b':
					runes = append(runes, '\b')
				case 'f':
					runes = append(runes, '\f')
				case 'n':
					runes = append(runes, '\n')
				case 'r':
					runes = append(runes, '\r')
				case 't':
					runes = append(runes, '\t')
				case 'v':
					runes = append(runes, '\v')
				case '\'':
					runes = append(runes, '\'')
				case '"':
					runes = append(runes, '"')
				case '?':
					runes = append(runes, '?')
				case 'u':
					unsafeRune, ok := lexHexDigits(l, 4, 4)
					if !ok {
						return l.invalidUniversalCharacterName()
					}
					if !isValidUniversalCharacterName(unsafeRune) {
						return l.invalidUniversalCharacterName()
					}
					runes = append(runes, unsafeRune)
				case 'U':
					unsafeRune, ok := lexHexDigits(l, 8, 8)
					if !ok {
						return l.invalidUniversalCharacterName()
					}
					if !isValidUniversalCharacterName(unsafeRune) {
						return l.invalidUniversalCharacterName()
					}
					runes = append(runes, unsafeRune)
				case 'x':
					unsafeRune, ok := lexHexDigits(l, 1, 2)
					if !ok {
						return l.invalidEscape()
					}
					if !isValidEscapedRune(unsafeRune) {
						return l.invalidEscape()
					}
					runes = append(runes, unsafeRune)
				default:
					if !isOctal(nextRune) {
						return l.invalidEscape()
					}
					l.backup()
					unsafeRune, ok := lexOctalDigits(l, 1, 3)
					if !ok {
						return l.invalidEscape()
					}
					if !isValidEscapedRune(unsafeRune) {
						return l.invalidEscape()
					}
					runes = append(runes, unsafeRune)
				}
			default:
				runes = append(runes, r)
			}
		}
	}
}

func lexStringASCIIPlist(state stateFn) stateFn {
	return func(l *lexer) stateFn {
		l.next()
		runes := []rune{}
		for {
			r := l.next()
			switch r {
			case eof, '\n', '\r':
				return l.unterminatedStringLiteral()
			case '"':
				l.emitValue(itemString, string(runes))
				return state
			case '\\':
				nextRune := l.next()
				switch nextRune {
				case eof:
					return l.unterminatedStringLiteral()
				case '\\':
					runes = append(runes, '\\')
				case 'a':
					runes = append(runes, '\a')
				case 'b':
					runes = append(runes, '\b')
				case 'f':
					runes = append(runes, '\f')
				case 'n':
					runes = append(runes, '\n')
				case 'r':
					runes = append(runes, '\r')
				case 't':
					runes = append(runes, '\t')
				case 'v':
					runes = append(runes, '\v')
				case '\'':
					runes = append(runes, '\'')
				case '"':
					runes = append(runes, '"')
				case '?':
					runes = append(runes, '?')
				case 'U':
					utf16CodeUnit, ok := lexHexDigits(l, 1, 4)
					if !ok {
						return l.invalidUTF16Escape()
					}
					if utf16.IsSurrogate(utf16CodeUnit) {
						highSurrogate := utf16CodeUnit
						if backslash := l.next(); backslash != '\\' {
							return l.invalidUTF16Escape()
						}
						if literalU := l.next(); literalU != 'U' {
							return l.invalidUTF16Escape()
						}
						lowSurrogate, ok := lexHexDigits(l, 4, 4)
						if !ok {
							return l.invalidUTF16Escape()
						}
						codePoint := utf16.DecodeRune(highSurrogate, lowSurrogate)
						if codePoint == '\uFFFD' {
							return l.invalidUTF16Escape()
						}
						runes = append(runes, codePoint)
					} else {
						runes = append(runes, utf16CodeUnit)
					}
				default:
					if !isOctal(nextRune) {
						return l.invalidEscape()
					}
					l.backup()
					unsafeRune, ok := lexOctalDigits(l, 3, 3)
					if !ok {
						return l.invalidEscape()
					}
					if !isValidEscapedRune(unsafeRune) {
						return l.invalidEscape()
					}
					runes = append(runes, unsafeRune)
				}
			default:
				runes = append(runes, r)
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

func lexBareString(state stateFn) stateFn {
	return func(l *lexer) stateFn {
		for {
			r := l.next()
			if !isASCIIPlistBareString(r) {
				if r != eof {
					l.backup()
				}
				l.emit(itemBareString)
				return state
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

func lexASCIIPlist(l *lexer) stateFn {
	for {
		if strings.HasPrefix(l.input[l.pos:], "/*") {
			return lexComment(lexASCIIPlist)
		}
		r := l.next()
		switch r {
		case eof:
			return l.eof()
		case '"':
			l.backup()
			return lexStringASCIIPlist(lexASCIIPlist)
		case ';':
			l.emit(itemSemicolon)
		case '=':
			l.emit(itemEqualSign)
		case '{':
			l.emit(itemBraceLeft)
		case '}':
			l.emit(itemBraceRight)
		case '(':
			l.emit(itemParenLeft)
		case ')':
			l.emit(itemParenRight)
		case ',':
			l.emit(itemComma)
		case '<':
			l.emit(itemLessThanSign)
		case '>':
			l.emit(itemGreaterThanSign)
		default:
			if isSpace(r) {
				l.backup()
				return lexSpaces(lexASCIIPlist)
			} else if isASCIIPlistBareString(r) {
				l.backup()
				return lexBareString(lexASCIIPlist)
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
