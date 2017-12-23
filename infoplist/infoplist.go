package infoplist

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

type InfoPlist map[string]string

type parser struct {
	decoder *xml.Decoder
}

func (p parser) NextToken() (xml.Token, error) {
	token, err := p.decoder.Token()
	return token, err
}

func (p parser) nextNonSpace() *xml.Token {
	for {
		token, err := p.NextToken()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			panic(err)
		}
		if _, ok := token.(xml.CharData); !ok {
			return &token
		}
	}
	return nil
}

func (p parser) recover(errp *error) {
	if r := recover(); r != nil {
		err, ok := r.(error)
		if !ok {
			panic("panicked without error")
		}
		*errp = err
	}
}

func (p parser) unexpected(what, expected interface{}) {
	panic(fmt.Errorf("unexpected %v; expected %v", what, expected))
}

func (p parser) unexpectedEOF(expected interface{}) {
	p.unexpected("EOF", expected)
}

func (p parser) expectXMLHeader() {
	expected := "<?xml?>"
	tokenp := p.nextNonSpace()
	if tokenp == nil {
		p.unexpectedEOF(expected)
	}
	token := *tokenp
	procInst, ok := token.(xml.ProcInst)
	if !ok ||
		procInst.Target != "xml" ||
		string(procInst.Inst) != `version="1.0" encoding="UTF-8"` {
		p.unexpected(token, expected)
	}
}

func (p parser) expectDocType() {
	expected := "<!DOCTYPE>"
	tokenp := p.nextNonSpace()
	if tokenp == nil {
		p.unexpectedEOF(expected)
	}
	token := *tokenp
	directive, ok := token.(xml.Directive)
	if !ok ||
		string(directive) != `DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd"` {
		p.unexpected(token, expected)
	}
}

func (p parser) assertStartElement(startElement xml.StartElement, localName string) {
	expected := "<" + localName + ">"
	if startElement.Name.Local != localName {
		p.unexpected(startElement, expected)
	}
}

func (p parser) expectStartElement(localName string) {
	expected := "<" + localName + ">"
	tokenp := p.nextNonSpace()
	if tokenp == nil {
		p.unexpectedEOF(expected)
	}
	token := *tokenp
	startElement, ok := token.(xml.StartElement)
	if !ok {
		p.unexpected(token, expected)
	}
	p.assertStartElement(startElement, localName)
}

func (p parser) assertEndElement(endElement xml.EndElement, localName string) {
	expected := "</" + localName + ">"
	if endElement.Name.Local != localName {
		p.unexpected(endElement, expected)
	}
}

func (p parser) expectEndElement(localName string) {
	expected := "</" + localName + ">"
	tokenp := p.nextNonSpace()
	if tokenp == nil {
		p.unexpectedEOF(expected)
	}
	token := *tokenp
	endElement, ok := token.(xml.EndElement)
	if !ok {
		p.unexpected(token, expected)
	}
	p.assertEndElement(endElement, localName)
}

func (p parser) expectCharData() string {
	token, err := p.NextToken()
	if err != nil {
		panic(err)
	}
	charData, ok := token.(xml.CharData)
	if !ok {
		p.unexpected(token, "CharData")
	}
	return string(charData.Copy())
}

func (p parser) parseDictValue() *string {
	expected := "<string>"

	tokenp := p.nextNonSpace()
	if tokenp == nil {
		p.unexpectedEOF(expected)
	}
	token := *tokenp
	startElement, ok := token.(xml.StartElement)
	if !ok {
		p.unexpected(token, expected)
	}
	if startElement.Name.Local != "string" {
		err := p.decoder.Skip()
		if err != nil {
			panic(err)
		}
		return nil
	}

	token, err := p.NextToken()
	if err != nil {
		panic(err)
	}
	switch v := token.(type) {
	case xml.CharData:
		value := string(v.Copy())
		p.expectEndElement("string")
		return &value
	case xml.EndElement:
		if v.Name.Local != "string" {
			p.unexpected(token, "</string>")
		}
		value := ""
		return &value
	}
	return nil
}

func (p parser) parseDict() InfoPlist {
	p.expectStartElement("dict")
	out := InfoPlist{}
Loop:
	for {
		expected := "<key> or </dict>"
		tokenp := p.nextNonSpace()
		if tokenp == nil {
			p.unexpectedEOF(expected)
		}
		token := *tokenp
		switch v := token.(type) {
		case xml.StartElement:
			p.assertStartElement(v, "key")
			key := p.expectCharData()
			p.expectEndElement("key")
			value := p.parseDictValue()
			if value != nil {
				out[key] = *value
			}
		case xml.EndElement:
			p.assertEndElement(v, "dict")
			break Loop
		default:
			p.unexpected(token, expected)
		}
	}
	return out
}

func (p parser) parsePlist() InfoPlist {
	p.expectStartElement("plist")
	out := p.parseDict()
	p.expectEndElement("plist")
	return out
}

func (p parser) expectEOF() {
	tokenp := p.nextNonSpace()
	if tokenp != nil {
		p.unexpected(*tokenp, "EOF")
	}
}

func (p parser) parse() InfoPlist {
	p.expectXMLHeader()
	p.expectDocType()
	out := p.parsePlist()
	p.expectEOF()
	return out
}

func ParseInfoPlist(src string) (out InfoPlist, err error) {
	reader := strings.NewReader(src)
	decoder := xml.NewDecoder(reader)
	p := parser{decoder}
	defer p.recover(&err)
	out = p.parse()
	return out, err
}
