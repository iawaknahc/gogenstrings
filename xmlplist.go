package main

import (
	"bytes"
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

type xmlPlistParser struct {
	decoder   *xml.Decoder
	offset    int
	filepath  string
	lineColer LineColer
}

func (p *xmlPlistParser) nextToken() (xml.Token, error) {
	p.offset = int(p.decoder.InputOffset())
	token, err := p.decoder.Token()
	return token, err
}

func (p *xmlPlistParser) nextNonSpace() xml.Token {
	for {
		token, err := p.nextToken()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			panic(err)
		}
		charData, ok := token.(xml.CharData)
		if !ok {
			return token
		}
		if len(bytes.TrimSpace([]byte(charData))) > 0 {
			return token
		}
	}
}

func (p *xmlPlistParser) recover(errp *error) {
	if r := recover(); r != nil {
		err, ok := r.(error)
		if !ok {
			panic("panicked without error")
		}
		*errp = err
	}
}

func (p *xmlPlistParser) unexpected(token, expected xml.Token) {
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

func (p *xmlPlistParser) unexpectedEOF(expected xml.Token) {
	p.unexpected(nil, expected)
}

func (p *xmlPlistParser) expectXMLHeader() {
	expected := xml.ProcInst{
		Target: "xml",
	}
	token := p.nextNonSpace()
	if token == nil {
		p.unexpectedEOF(expected)
	}
	procInst, ok := token.(xml.ProcInst)
	if !ok ||
		procInst.Target != "xml" ||
		string(procInst.Inst) != `version="1.0" encoding="UTF-8"` {
		p.unexpected(token, expected)
	}
}

func (p *xmlPlistParser) expectDocType() {
	expected := xml.Directive([]byte("DOCTYPE"))
	token := p.nextNonSpace()
	if token == nil {
		p.unexpectedEOF(expected)
	}
	directive, ok := token.(xml.Directive)
	if !ok ||
		string(directive) != `DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd"` {
		p.unexpected(token, expected)
	}
}

func (p *xmlPlistParser) assertStartElement(startElement xml.StartElement, localName string) {
	expected := makeStartElement(localName)
	if startElement.Name.Local != localName {
		p.unexpected(startElement, expected)
	}
}

func (p *xmlPlistParser) expectStartElement(localName string) {
	expected := makeStartElement(localName)
	token := p.nextNonSpace()
	if token == nil {
		p.unexpectedEOF(expected)
	}
	startElement, ok := token.(xml.StartElement)
	if !ok {
		p.unexpected(token, expected)
	}
	p.assertStartElement(startElement, localName)
}

func (p *xmlPlistParser) assertEndElement(endElement xml.EndElement, localName string) {
	expected := makeEndElement(localName)
	if endElement.Name.Local != localName {
		p.unexpected(endElement, expected)
	}
}

func (p *xmlPlistParser) expectEndElement(localName string) {
	expected := makeEndElement(localName)
	token := p.nextNonSpace()
	if token == nil {
		p.unexpectedEOF(expected)
	}
	endElement, ok := token.(xml.EndElement)
	if !ok {
		p.unexpected(token, expected)
	}
	p.assertEndElement(endElement, localName)
}

func (p *xmlPlistParser) expectCharData() string {
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

func (p *xmlPlistParser) parseDictValue() *string {
	expected := makeStartElement("string")

	token := p.nextNonSpace()
	if token == nil {
		p.unexpectedEOF(expected)
	}
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

func (p *xmlPlistParser) parseDict() infoPlist {
	p.expectStartElement("dict")
	out := infoPlist{}
Loop:
	for {
		expected := "<key> or </dict>"
		token := p.nextNonSpace()
		if token == nil {
			p.unexpectedEOF(expected)
		}
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

func (p *xmlPlistParser) parsePlist() infoPlist {
	p.expectStartElement("plist")
	out := p.parseDict()
	p.expectEndElement("plist")
	return out
}

func (p *xmlPlistParser) expectEOF() {
	token := p.nextNonSpace()
	if token != nil {
		p.unexpected(token, nil)
	}
}

func (p *xmlPlistParser) parse() infoPlist {
	p.expectXMLHeader()
	p.expectDocType()
	out := p.parsePlist()
	p.expectEOF()
	return out
}

func parseInfoPlist(src, filepath string) (out infoPlist, err error) {
	reader := strings.NewReader(src)
	decoder := xml.NewDecoder(reader)
	p := xmlPlistParser{
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
