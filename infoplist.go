package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

type infoPlist map[string]string

func isLocalizableKey(key string) bool {
	if strings.HasSuffix(key, "UsageDescription") {
		return true
	}
	return key == "CFBundleDisplayName"
}

func isValueVariable(value string) bool {
	if strings.HasPrefix(value, "$(") && strings.HasSuffix(value, ")") {
		return true
	}
	return false
}

func isKeyValueLocalizable(key, value string) bool {
	return isLocalizableKey(key) && !isValueVariable(value)
}

func (p infoPlist) localizable() infoPlist {
	out := infoPlist{}
	for key, value := range p {
		if isKeyValueLocalizable(key, value) {
			out[key] = value
		}
	}
	return out
}

func (p infoPlist) toEntryMap() entryMap {
	out := entryMap{}
	for key, value := range p {
		out[key] = entry{
			key:   key,
			value: value,
		}
	}
	return out
}

type parser struct {
	decoder   *xml.Decoder
	offset    int
	filepath  string
	lineColer LineColer
}

func (p *parser) nextToken() (xml.Token, error) {
	p.offset = int(p.decoder.InputOffset())
	token, err := p.decoder.Token()
	return token, err
}

func (p *parser) nextNonSpace() *xml.Token {
	for {
		token, err := p.nextToken()
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
}

func (p *parser) recover(errp *error) {
	if r := recover(); r != nil {
		err, ok := r.(error)
		if !ok {
			panic("panicked without error")
		}
		*errp = err
	}
}

func (p *parser) unexpected(token, expected xml.Token) {
	startLine, startCol := p.lineColer.LineCol(p.offset)
	message := fmt.Sprintf(
		"unexpected %v; expected %v",
		tokenToString(token),
		tokenToString(expected),
	)
	err := makeErrFileLineCol(
		p.filepath,
		startLine,
		startCol,
		message,
	)
	panic(err)
}

func (p *parser) unexpectedEOF(expected xml.Token) {
	p.unexpected(nil, expected)
}

func (p *parser) expectXMLHeader() {
	expected := xml.ProcInst{
		Target: "xml",
	}
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

func (p *parser) expectDocType() {
	expected := xml.Directive([]byte("DOCTYPE"))
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

func (p *parser) assertStartElement(startElement xml.StartElement, localName string) {
	expected := makeStartElement(localName)
	if startElement.Name.Local != localName {
		p.unexpected(startElement, expected)
	}
}

func (p *parser) expectStartElement(localName string) {
	expected := makeStartElement(localName)
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

func (p *parser) assertEndElement(endElement xml.EndElement, localName string) {
	expected := makeEndElement(localName)
	if endElement.Name.Local != localName {
		p.unexpected(endElement, expected)
	}
}

func (p *parser) expectEndElement(localName string) {
	expected := makeEndElement(localName)
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

func (p *parser) expectCharData() string {
	token, err := p.nextToken()
	if err != nil {
		panic(err)
	}
	charData, ok := token.(xml.CharData)
	if !ok {
		p.unexpected(token, xml.CharData{})
	}
	return string(charData.Copy())
}

func (p *parser) parseDictValue() *string {
	expected := makeStartElement("string")

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

	token, err := p.nextToken()
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
			p.unexpected(token, makeEndElement("string"))
		}
		value := ""
		return &value
	}
	return nil
}

func (p *parser) parseDict() infoPlist {
	p.expectStartElement("dict")
	out := infoPlist{}
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

func (p *parser) parsePlist() infoPlist {
	p.expectStartElement("plist")
	out := p.parseDict()
	p.expectEndElement("plist")
	return out
}

func (p *parser) expectEOF() {
	tokenp := p.nextNonSpace()
	if tokenp != nil {
		p.unexpected(*tokenp, nil)
	}
}

func (p *parser) parse() infoPlist {
	p.expectXMLHeader()
	p.expectDocType()
	out := p.parsePlist()
	p.expectEOF()
	return out
}

func parseInfoPlist(src, filepath string) (out infoPlist, err error) {
	reader := strings.NewReader(src)
	decoder := xml.NewDecoder(reader)
	p := parser{
		decoder:   decoder,
		filepath:  filepath,
		lineColer: NewLineColer(src),
	}
	defer p.recover(&err)
	out = p.parse()
	return out, err
}

func truncateBytes(b []byte) string {
	s := string(b)
	if len(s) >= 20 {
		return s[0:20] + "..."
	}
	return s
}

func tokenToString(token xml.Token) string {
	if token == nil {
		return "EOF"
	}
	switch v := token.(type) {
	case xml.StartElement:
		return "<" + v.Name.Local + ">"
	case xml.EndElement:
		return "</" + v.Name.Local + ">"
	case xml.CharData:
		return "CharData"
	case xml.Comment:
		return "Comment"
	case xml.ProcInst:
		return "<?" + v.Target + "?>"
	case xml.Directive:
		return "<!" + truncateBytes(v) + ">"
	case string:
		return v
	}
	return ""
}

func makeStartElement(localName string) xml.StartElement {
	return xml.StartElement{
		Name: xml.Name{
			Local: localName,
		},
	}
}

func makeEndElement(localName string) xml.EndElement {
	return xml.EndElement{
		Name: xml.Name{
			Local: localName,
		},
	}
}
