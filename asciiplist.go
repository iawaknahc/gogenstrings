package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"unicode"
	"unicode/utf16"

	"github.com/iawaknahc/gogenstrings/errors"
)

// ASCIIPlistNode represents a node in plist.
// The zero value is not safe to use.
type ASCIIPlistNode struct {
	// Value stores the actual value.
	// The mapping is as follows:
	// "string" or string             -> string
	// <abcdef1234567890>             -> []byte
	// (array)                        -> []ASCIIPlistNode
	// {_=dict;}                      -> ASCIIPlistDict
	Value interface{}
	// Line is the line number.
	Line int
	// Col is the column number.
	Col int
	// CommentBefore is the comment before this node.
	CommentBefore string
	// CommentAfter is the comment after this node.
	CommentAfter string
}

// ASCIIPlistDict represents a dict preserving key order.
type ASCIIPlistDict struct {
	Keys []ASCIIPlistNode
	Map  map[ASCIIPlistNode]ASCIIPlistNode
}

type annotatedItem struct {
	item       lexItem
	comment    lexItem
	hasComment bool
}

func (v annotatedItem) getComment() string {
	if !v.hasComment {
		return ""
	}
	return v.comment.Value
}

func (v annotatedItem) canHaveCommentBefore() bool {
	if !v.hasComment {
		return false
	}
	switch v.item.Type {
	case itemString, itemBareString, itemBraceLeft, itemLessThanSign, itemParenLeft:
		return true
	}
	return false
}

// Flatten turns the receiver to Go value.
func (v ASCIIPlistNode) Flatten() interface{} {
	switch x := v.Value.(type) {
	case string:
		return x
	case []byte:
		return x
	case []ASCIIPlistNode:
		out := make([]interface{}, len(x))
		for i, value := range x {
			out[i] = value.Flatten()
		}
		return out
	case ASCIIPlistDict:
		out := make(map[string]interface{}, len(x.Map))
		for key, value := range x.Map {
			out[key.Value.(string)] = value.Flatten()
		}
		return out
	}
	panic(fmt.Errorf("unreachable"))
}

type asciiPlistParser struct {
	filepath  string
	lexer     *lexer
	peekCount int
	token     [2]annotatedItem
}

func (p *asciiPlistParser) next() annotatedItem {
	if p.peekCount > 0 {
		p.peekCount--
	} else {
		item := p.lexer.nextItem()
		aitem := annotatedItem{
			item: item,
		}
		p.token[0] = aitem
	}
	return p.token[p.peekCount]
}

func (p *asciiPlistParser) backup(t0 annotatedItem) {
	p.token[p.peekCount] = t0
	p.peekCount++
}

func (p *asciiPlistParser) backup2(t0, t1 annotatedItem) {
	p.token[0] = t0
	p.token[1] = t1
	p.peekCount = 2
}

func (p *asciiPlistParser) nextNonSpace() annotatedItem {
	var aitem annotatedItem
	var comment lexItem
	hasComment := false
Loop:
	for {
		aitem = p.next()
		switch aitem.item.Type {
		case itemSpaces:
			break
		case itemComment:
			hasComment = true
			comment = aitem.item
		default:
			if hasComment {
				aitem.comment = comment
				aitem.hasComment = hasComment
			}
			break Loop
		}
	}
	return aitem
}

func (p *asciiPlistParser) peekNonSpace() annotatedItem {
	aitem := p.nextNonSpace()
	p.backup(aitem)
	return aitem
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

func (p *asciiPlistParser) expect(expected itemType) annotatedItem {
	aitem := p.nextNonSpace()
	if aitem.item.Type != expected {
		p.unexpected(aitem)
	}
	return aitem
}

func (p *asciiPlistParser) unexpected(aitem annotatedItem) {
	if aitem.item.Type == itemError {
		panic(aitem.item.Err)
	} else {
		panic(aitem.item.unexpectedTokenErr())
	}
}

func (p *asciiPlistParser) parseValue() (out ASCIIPlistNode) {
	token := p.nextNonSpace()
	switch token.item.Type {
	case itemString, itemBareString:
		p.backup(token)
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

func (p *asciiPlistParser) parseString() (out ASCIIPlistNode) {
	token := p.nextNonSpace()
	switch token.item.Type {
	case itemString, itemBareString:
		out.Value = token.item.Value
		out.Line = token.item.StartLine
		out.Col = token.item.StartCol
		out.CommentBefore = token.getComment()
		if nextToken := p.peekNonSpace(); !nextToken.canHaveCommentBefore() {
			out.CommentAfter = nextToken.getComment()
		}
	default:
		p.unexpected(token)
	}
	return
}

func (p *asciiPlistParser) parseDict(startToken annotatedItem, terminatingType itemType) (out ASCIIPlistNode) {
	seenKeys := make(map[string]bool)
	outValue := ASCIIPlistDict{
		Keys: []ASCIIPlistNode{},
		Map:  make(map[ASCIIPlistNode]ASCIIPlistNode),
	}
	out.Line = startToken.item.StartLine
	out.Col = startToken.item.StartCol
	out.CommentBefore = startToken.getComment()
	for {
		token := p.nextNonSpace()
		if token.item.Type == terminatingType {
			if nextToken := p.peekNonSpace(); !nextToken.canHaveCommentBefore() {
				out.CommentAfter = nextToken.getComment()
			}
			out.Value = outValue
			return
		}
		p.backup(token)
		keyValue := p.parseString()
		p.expect(itemEqualSign)
		valueValue := p.parseValue()
		p.expect(itemSemicolon)
		key := keyValue.Value.(string)
		if seen := seenKeys[key]; seen {
			panic(errors.FileLineCol(
				p.filepath,
				keyValue.Line,
				keyValue.Col,
				fmt.Sprintf("duplicated key `%v`", key),
			))
		}
		seenKeys[key] = true
		outValue.Keys = append(outValue.Keys, keyValue)
		outValue.Map[keyValue] = valueValue
	}
}

func (p *asciiPlistParser) parseArray(startToken annotatedItem) (out ASCIIPlistNode) {
	outValue := []ASCIIPlistNode{}
	out.Line = startToken.item.StartLine
	out.Col = startToken.item.StartCol
	out.CommentBefore = startToken.getComment()
	first := true
	for {
		token := p.nextNonSpace()
		if token.item.Type == itemParenRight {
			out.Value = outValue
			if nextToken := p.peekNonSpace(); !nextToken.canHaveCommentBefore() {
				out.CommentAfter = nextToken.getComment()
			}
			return
		}
		p.backup(token)
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

func (p *asciiPlistParser) parseData(startToken annotatedItem) (out ASCIIPlistNode) {
	buf := bytes.Buffer{}
	out.Line = startToken.item.StartLine
	out.Col = startToken.item.StartCol
	out.CommentBefore = startToken.getComment()
	for {
		token := p.nextNonSpace()
		switch token.item.Type {
		case itemGreaterThanSign:
			src := buf.Bytes()
			dst := make([]byte, hex.DecodedLen(len(src)))
			_, err := hex.Decode(dst, src)
			if err != nil {
				panic("impossible")
			}
			out.Value = dst
			if nextToken := p.peekNonSpace(); !nextToken.canHaveCommentBefore() {
				out.CommentAfter = nextToken.getComment()
			}
			return
		case itemBareString:
			if !isASCIIPlistHex(token.item.Value) {
				p.unexpected(token)
			}
			buf.WriteString(token.item.Value)
		default:
			p.unexpected(token)
		}
	}
}

func (p *asciiPlistParser) parse() (out ASCIIPlistNode, err error) {
	defer p.recover(&err)
	token := p.nextNonSpace()
	switch token.item.Type {
	case itemEOF:
		p.backup(token)
		out = p.parseDict(token, itemEOF)
	case itemString, itemBareString:
		nextToken := p.nextNonSpace()
		switch nextToken.item.Type {
		case itemEOF:
			p.backup2(nextToken, token)
			out = p.parseString()
		case itemEqualSign:
			p.backup2(nextToken, token)
			out = p.parseDict(token, itemEOF)
		default:
			p.unexpected(nextToken)
		}
	default:
		p.backup(token)
		out = p.parseValue()
	}
	p.expect(itemEOF)
	return
}

func parseASCIIPlist(src, filepath string) (ASCIIPlistNode, error) {
	l := newLexer(src, filepath, lexASCIIPlist)
	p := &asciiPlistParser{
		filepath: filepath,
		lexer:    &l,
	}
	return p.parse()
}

// PrintASCIIPlistString turns a string to a string literal
func PrintASCIIPlistString(s string) string {
	buf := bytes.Buffer{}
	buf.WriteRune('"')
	for _, r := range s {
		switch r {
		case '\\':
			buf.WriteString(`\\`)
		case '\a':
			buf.WriteString(`\a`)
		case '\b':
			buf.WriteString(`\b`)
		case '\f':
			buf.WriteString(`\f`)
		case '\n':
			buf.WriteString(`\n`)
		case '\r':
			buf.WriteString(`\r`)
		case '\t':
			buf.WriteString(`\t`)
		case '\v':
			buf.WriteString(`\v`)
		case '"':
			buf.WriteString(`\"`)
		default:
			if unicode.IsPrint(r) {
				buf.WriteRune(r)
			} else {
				if r < 0x10000 {
					buf.WriteString(fmt.Sprintf(`\U%04X`, r))
				} else {
					r1, r2 := utf16.EncodeRune(r)
					buf.WriteString(fmt.Sprintf(`\U%04X\U%04X`, r1, r2))
				}
			}
		}
	}
	buf.WriteRune('"')
	return buf.String()
}
