package main

import (
	"fmt"
	"path"

	"github.com/iawaknahc/gogenstrings/errors"
)

type routineCall struct {
	filepath  string
	startLine int
	startCol  int
	key       string
	comment   string
}

type routineCallSlice []routineCall

func (p routineCallSlice) toMap() (map[string]routineCall, error) {
	out := map[string]routineCall{}
	for _, call := range p {
		// Validate every call has non-empty key
		if call.key == "" {
			return nil, errors.FileLineCol(
				call.filepath,
				call.startLine,
				call.startCol,
				"routine call has empty key",
			)
		}

		// Validate calls having the same key has the same comment
		existingCall, ok := out[call.key]
		if ok {
			if call.comment != existingCall.comment {
				return nil, errors.FileLineCol(
					call.filepath,
					call.startLine,
					call.startCol,
					fmt.Sprintf("routine call `%v` has different comment", call.key),
				)
			}
		}

		out[call.key] = call
	}
	return out, nil
}

func parseRoutineCalls(src, routineName, filepath string) (routineCallSlice, error) {
	var lexString func(stateFn) stateFn
	switch path.Ext(filepath) {
	case ".swift":
		lexString = lexStringSwift
	case ".m", ".h":
		lexString = lexStringObjc
	default:
		return nil, errors.File(
			filepath,
			"unknown file type",
		)
	}
	l := newLexerWithString(src, filepath, lexString, lexRoutineCall)
	p := &routineCallParser{
		filepath:    filepath,
		routineName: routineName,
		lexer:       &l,
	}
	return p.parse()
}

type routineCallParser struct {
	filepath    string
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
		panic(item.unexpectedTokenErr())
	}
}

func (p *routineCallParser) parse() (output routineCallSlice, outerr error) {
	defer p.recover(&outerr)
	for {
		token := p.nextNonSpace()
		if token.Type == itemEOF {
			break
		}
		if token.Type == itemError {
			return nil, token.Err
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
			filepath:  p.filepath,
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
		output += token.Value
	} else if token.Type == itemString {
		output += token.Value
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
			output += token.Value
		} else if !atSign && token.Type == itemString {
			output += token.Value
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
