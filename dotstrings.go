package main

type dotStringsParser struct {
	filepath string
	lexer    *lexer
}

func (p *dotStringsParser) parse() (output entries, err error) {
	defer p.recover(&err)
	for {
		token := p.nextNonSpace()
		if token.Type == itemEOF {
			break
		}
		var key lexItem
		var comment string

		if token.Type == itemComment {
			comment = getComment(token.Value)
			key = p.expect(itemString)
		} else if token.Type == itemString {
			comment = ""
			key = token
		} else {
			p.unexpected(token)
		}

		p.expect(itemEqualSign)
		value := p.expect(itemString)
		p.expect(itemSemicolon)
		ls := entry{
			filepath:  p.filepath,
			startLine: key.StartLine,
			startCol:  key.StartCol,
			comment:   comment,
			key:       getStringValue(key.Value),
			value:     getStringValue(value.Value),
		}
		output = append(output, ls)
	}
	return output, nil
}

func (p *dotStringsParser) nextNonSpace() lexItem {
	for {
		item := p.lexer.nextItem()
		if item.Type != itemSpaces {
			return item
		}
	}
}

func (p *dotStringsParser) recover(errp *error) {
	if r := recover(); r != nil {
		err, ok := r.(error)
		if !ok {
			panic("panicked without error")
		}
		*errp = err
	}
}

func (p *dotStringsParser) expect(expected itemType) lexItem {
	item := p.nextNonSpace()
	if item.Type != expected {
		p.unexpected(item)
	}
	return item
}

func (p *dotStringsParser) unexpected(item lexItem) {
	if item.Type == itemError {
		panic(item.Err)
	} else {
		panic(item.unexpectedTokenErr())
	}
}

func parseDotStrings(src, filepath string) (entries, error) {
	l := newLexer(src, filepath, lexEntry)
	p := &dotStringsParser{
		filepath: filepath,
		lexer:    &l,
	}
	return p.parse()
}
