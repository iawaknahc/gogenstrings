package main

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/iawaknahc/gogenstrings/errors"
)

// ASCIIPlistValue represents a value in plist.
type ASCIIPlistValue struct {
	// Value stores the actual value.
	// The mapping is as follows:
	// "string" or string             -> string
	// <abcdef1234567890>             -> []byte
	// (array)                        -> []interface{}
	// {_=dict;}                      -> map[string]interface{}
	Value interface{}
	// Line is the line number.
	Line int
	// Col is the column number.
	Col int
}

func (v ASCIIPlistValue) String() string {
	switch x := v.Value.(type) {
	case string:
		return fmt.Sprintf("%v:%v %q", v.Line, v.Col, x)
	case []byte:
		return fmt.Sprintf("%v:%v %v", v.Line, v.Col, x)
	case []interface{}:
		return fmt.Sprintf("%v:%v %v", v.Line, v.Col, x)
	case map[string]interface{}:
		return fmt.Sprintf("%v:%v %v", v.Line, v.Col, x)
	}
	panic(fmt.Errorf("unreachable"))
}

// Flatten turns the receiver to Go value.
func (v ASCIIPlistValue) Flatten() interface{} {
	switch x := v.Value.(type) {
	case string:
		return x
	case []byte:
		return x
	case []interface{}:
		out := make([]interface{}, len(x))
		for i, value := range x {
			out[i] = value.(ASCIIPlistValue).Flatten()
		}
		return out
	case map[string]interface{}:
		out := make(map[string]interface{}, len(x))
		for key, value := range x {
			out[key] = value.(ASCIIPlistValue).Flatten()
		}
		return out
	}
	panic(fmt.Errorf("unreachable"))
}

type asciiPlistParser struct {
	filepath  string
	lexer     *lexer
	peekCount int
	token     [2]lexItem
}

func (p *asciiPlistParser) next() lexItem {
	if p.peekCount > 0 {
		p.peekCount--
	} else {
		p.token[0] = p.lexer.nextItem()
	}
	return p.token[p.peekCount]
}

func (p *asciiPlistParser) backup() {
	p.peekCount++
}

func (p *asciiPlistParser) backup2(t1 lexItem) {
	p.token[1] = t1
	p.peekCount = 2
}

func (p *asciiPlistParser) nextNonSpace() lexItem {
	for {
		item := p.next()
		if item.Type != itemSpaces {
			return item
		}
	}
}

func (p *asciiPlistParser) recover(errp *error) {
	if r := recover(); r != nil {
		err, ok := r.(error)
		if !ok {
			panic("panicked without error")
		}
		*errp = err
	}
}

func (p *asciiPlistParser) expect(expected itemType) lexItem {
	item := p.nextNonSpace()
	if item.Type != expected {
		p.unexpected(item)
	}
	return item
}

func (p *asciiPlistParser) unexpected(item lexItem) {
	if item.Type == itemError {
		panic(item.Err)
	} else {
		panic(item.unexpectedTokenErr())
	}
}

func (p *asciiPlistParser) parseValue() (out ASCIIPlistValue) {
	token := p.nextNonSpace()
	switch token.Type {
	case itemString, itemBareString:
		p.backup()
		out = p.parseString()
	case itemBraceLeft:
		out = p.parseDict(token, itemBraceRight)
	case itemLessThanSign:
		out = p.parseData(token)
	case itemParenLeft:
		out = p.parseArray(token)
	default:
		p.unexpected(token)
	}
	return
}

func (p *asciiPlistParser) parseString() (out ASCIIPlistValue) {
	token := p.nextNonSpace()
	switch token.Type {
	case itemString, itemBareString:
		value := ""
		if token.Type == itemString {
			value = getStringValue(token.Value)
		} else {
			value = token.Value
		}
		out.Value = value
		out.Line = token.StartLine
		out.Col = token.StartCol
	default:
		p.unexpected(token)
	}
	return
}

func (p *asciiPlistParser) parseDict(startToken lexItem, terminatingType itemType) (out ASCIIPlistValue) {
	outValue := make(map[string]interface{})
	out.Value = outValue
	out.Line = startToken.StartLine
	out.Col = startToken.StartCol
	for {
		token := p.nextNonSpace()
		if token.Type == terminatingType {
			return
		}
		p.backup()
		keyValue := p.parseString()
		p.expect(itemEqualSign)
		valueValue := p.parseValue()
		p.expect(itemSemicolon)
		key := keyValue.Value.(string)
		if _, ok := outValue[key]; ok {
			panic(errors.FileLineCol(
				p.filepath,
				keyValue.Line,
				keyValue.Col,
				fmt.Sprintf("duplicated key `%v`", key),
			))
		}
		outValue[key] = valueValue
	}
}

func (p *asciiPlistParser) parseArray(startToken lexItem) (out ASCIIPlistValue) {
	outValue := []interface{}{}
	out.Line = startToken.StartLine
	out.Col = startToken.StartCol
	first := true
	for {
		token := p.nextNonSpace()
		if token.Type == itemParenRight {
			out.Value = outValue
			return
		}
		p.backup()
		if !first {
			p.expect(itemComma)
		}
		valueValue := p.parseValue()
		if first {
			first = false
		}
		outValue = append(outValue, valueValue)
	}
}

func isASCIIPlistHex(s string) bool {
	if s == "" {
		return false
	}
	length := 0
	for _, r := range s {
		length++
		if !isHex(r) {
			return false
		}
	}
	return length%2 == 0
}

func (p *asciiPlistParser) parseData(startToken lexItem) (out ASCIIPlistValue) {
	buf := bytes.Buffer{}
	out.Line = startToken.StartLine
	out.Col = startToken.StartCol
	for {
		token := p.nextNonSpace()
		switch token.Type {
		case itemGreaterThanSign:
			src := buf.Bytes()
			dst := make([]byte, hex.DecodedLen(len(src)))
			_, err := hex.Decode(dst, src)
			if err != nil {
				panic("impossible")
			}
			out.Value = dst
			return
		case itemBareString:
			if !isASCIIPlistHex(token.Value) {
				p.unexpected(token)
			}
			buf.WriteString(token.Value)
		default:
			p.unexpected(token)
		}
	}
}

func (p *asciiPlistParser) parse() (out ASCIIPlistValue, err error) {
	defer p.recover(&err)
	token := p.nextNonSpace()
	if token.Type == itemEOF {
		out.Value = make(map[string]interface{})
		return
	}
	switch token.Type {
	case itemString, itemBareString:
		nextToken := p.nextNonSpace()
		switch nextToken.Type {
		case itemEOF:
			p.backup2(token)
			out = p.parseString()
		case itemEqualSign:
			p.backup2(token)
			out = p.parseDict(token, itemEOF)
		default:
			p.unexpected(nextToken)
		}
	default:
		p.backup()
		out = p.parseValue()
	}
	p.expect(itemEOF)
	return
}

func parseASCIIPlist(src, filepath string) (ASCIIPlistValue, error) {
	l := newLexer(src, filepath, lexASCIIPlist)
	p := &asciiPlistParser{
		filepath: filepath,
		lexer:    &l,
	}
	return p.parse()
}
