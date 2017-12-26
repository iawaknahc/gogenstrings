package main

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/iawaknahc/gogenstrings/errors"
)

// ASCIIPlistNode represents a node in plist.
// The zero value is not safe to use.
type ASCIIPlistNode struct {
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
	// CommentBefore is the comment before this node.
	CommentBefore string
	// CommentAfter is the comment after this node.
	CommentAfter string
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
	return getComment(v.comment.Value)
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
	case map[ASCIIPlistNode]ASCIIPlistNode:
		out := make(map[string]interface{}, len(x))
		for key, value := range x {
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
		value := ""
		if token.item.Type == itemString {
			value = getStringValue(token.item.Value)
		} else {
			value = token.item.Value
		}
		out.Value = value
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
	outValue := make(map[ASCIIPlistNode]ASCIIPlistNode)
	out.Value = outValue
	out.Line = startToken.item.StartLine
	out.Col = startToken.item.StartCol
	out.CommentBefore = startToken.getComment()
	for {
		token := p.nextNonSpace()
		if token.item.Type == terminatingType {
			if nextToken := p.peekNonSpace(); !nextToken.canHaveCommentBefore() {
				out.CommentAfter = nextToken.getComment()
			}
			return
		}
		p.backup(token)
		keyValue := p.parseString()
		p.expect(itemEqualSign)
		valueValue := p.parseValue()
		p.expect(itemSemicolon)
		if _, ok := outValue[keyValue]; ok {
			panic(errors.FileLineCol(
				p.filepath,
				keyValue.Line,
				keyValue.Col,
				fmt.Sprintf("duplicated key `%v`", keyValue.Value.(string)),
			))
		}
		outValue[keyValue] = valueValue
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
