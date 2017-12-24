package main

import (
	"fmt"
)

type routineCall struct {
	startLine int
	startCol  int
	key       string
	comment   string
	// the first path this routine call is found
	path string
}

func parseRoutineCall(src, routineName string) ([]routineCall, error) {
	l := newLexer(src, lexRoutineCall)
	p := &routineCallParser{
		routineName: routineName,
		lexer:       &l,
	}
	return p.parse()
}

type routineCallParser struct {
	routineName string
	lexer       *lexer
	peekCount   int
	token       [1]lexItem
}

func (p *routineCallParser) next() lexItem {
	if p.peekCount > 0 {
		p.peekCount--
	} else {
		p.token[0] = p.lexer.nextItem()
	}
	return p.token[p.peekCount]
}

func (p *routineCallParser) backup() {
	p.peekCount++
}

func (p *routineCallParser) nextNonSpace() (item lexItem) {
	for {
		item = p.next()
		if item.Type != itemSpaces {
			break
		}
	}
	return item
}

func (p *routineCallParser) recover(errp *error) {
	if r := recover(); r != nil {
		err, ok := r.(error)
		if !ok {
			panic("panicked without error")
		}
		*errp = err
	}
}

func (p *routineCallParser) expect(expected itemType) lexItem {
	item := p.nextNonSpace()
	if item.Type != expected {
		p.unexpected(item)
	}
	return item
}

func (p *routineCallParser) unexpected(item lexItem) {
	if item.Type == itemError {
		panic(item.Err)
	} else {
		panic(fmt.Errorf("unexpected token %v", item))
	}
}

func (p *routineCallParser) parse() (output []routineCall, outerr error) {
	defer p.recover(&outerr)
	for {
		token := p.nextNonSpace()
		if token.Type == itemEOF {
			break
		}
		if token.Type != itemIdentifier || token.Value != p.routineName {
			continue
		}
		p.expect(itemParenLeft)
		key := p.parseString()
		p.expect(itemComma)
		p.parseFuncLabel()
		comment := p.parseString()
		p.expect(itemParenRight)
		rc := routineCall{
			startLine: token.StartLine,
			startCol:  token.StartCol,
			key:       key,
			comment:   comment,
		}
		output = append(output, rc)
	}
	return output, nil
}

func (p *routineCallParser) parseString() (output string) {
	atSign := false
	token := p.nextNonSpace()

	if token.Type == itemAtSign {
		atSign = true
		token = p.nextNonSpace()
		if token.Type != itemString {
			p.unexpected(token)
		}
		output += getStringValue(token.Value)
	} else if token.Type == itemString {
		output += getStringValue(token.Value)
	} else {
		p.unexpected(token)
	}

	for {
		token = p.nextNonSpace()
		if atSign && token.Type == itemAtSign {
			token = p.nextNonSpace()
			if token.Type != itemString {
				p.unexpected(token)
				break
			}
			output += getStringValue(token.Value)
		} else if !atSign && token.Type == itemString {
			output += getStringValue(token.Value)
		} else {
			p.backup()
			break
		}
	}

	return output
}

func (p *routineCallParser) parseFuncLabel() {
	token := p.nextNonSpace()
	if token.Type != itemIdentifier {
		p.backup()
		return
	}
	p.expect(itemColon)
}
