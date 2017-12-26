package xmlplist

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/iawaknahc/gogenstrings/errors"
	"github.com/iawaknahc/gogenstrings/linecol"
)

// Value represents a value in plist.
// The zero value is not safe to use.
type Value struct {
	// Value stores the actual value.
	// The mapping is as follows:
	// <string>  -> string
	// <real>    -> float64
	// <integer> -> int64
	// <true>    -> bool
	// <false>   -> bool
	// <date>    -> time.Time
	// <data>    -> []byte
	// <array>   -> []interface{}
	// <dict>    -> map[string]interface{}
	Value interface{}
	// Line is the line number.
	Line int
	// Col is the column number.
	Col int
}

func (v Value) String() string {
	switch x := v.Value.(type) {
	case string:
		return "<string>"
	case float64:
		return "<real>"
	case int64:
		return "<integer>"
	case bool:
		if x {
			return "<true>"
		}
		return "<false>"
	case time.Time:
		return "<date>"
	case []byte:
		return "<data>"
	case []interface{}:
		return "<array>"
	case map[string]interface{}:
		return "<dict>"
	}
	panic(fmt.Errorf("unreachable"))
}

// Flatten turns the receiver to Go value.
func (v Value) Flatten() interface{} {
	switch x := v.Value.(type) {
	case string:
		return x
	case float64:
		return x
	case int64:
		return x
	case bool:
		return x
	case time.Time:
		return x
	case []byte:
		return x
	case []interface{}:
		out := make([]interface{}, len(x))
		for i, value := range x {
			out[i] = value.(Value).Flatten()
		}
		return out
	case map[string]interface{}:
		out := make(map[string]interface{}, len(x))
		for key, value := range x {
			out[key] = value.(Value).Flatten()
		}
		return out
	}
	panic(fmt.Errorf("unreachable"))
}

func makeXMLPlistValue(value interface{}, line, col int) Value {
	return Value{
		Value: value,
		Line:  line,
		Col:   col,
	}
}

const (
	anyPlistValue string = "one of <string>, <real>, <integer>, <true>, <false>, <date>, <data>, <array>, <dict>"
)

type xmlPlistParser struct {
	decoder   *xml.Decoder
	offset    int
	filepath  string
	lineColer linecol.LineColer
}

func (p *xmlPlistParser) nextToken() xml.Token {
	p.offset = int(p.decoder.InputOffset())
	token, err := p.decoder.Token()
	if err != nil {
		if err == io.EOF {
			return nil
		}
		if syntaxErr, ok := err.(*xml.SyntaxError); ok {
			line, col := p.lineColer.LineCol(p.offset)
			panic(errors.FileLineCol(
				p.filepath,
				line,
				col,
				syntaxErr.Msg,
			))
		} else {
			panic(err)
		}
	}
	return token
}

func (p *xmlPlistParser) nextNonSpace() xml.Token {
	for {
		token := p.nextToken()
		if token == nil {
			return nil
		}
		// Ignore comment
		if _, ok := token.(xml.Comment); ok {
			continue
		}
		// Ignore whitespace CharData
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
	panic(errors.FileLineCol(
		p.filepath,
		startLine,
		startCol,
		message,
	))
}

func (p *xmlPlistParser) unexpectedEOF(expected xml.Token) {
	p.unexpected(nil, expected)
}

func (p *xmlPlistParser) expectXMLHeader() {
	expected := xml.ProcInst{
		Target: "xml",
	}
	token := p.nextToken()
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

func (p *xmlPlistParser) expectStartElement(skipSpace bool, expected xml.Token) xml.StartElement {
	var token xml.Token
	if skipSpace {
		token = p.nextNonSpace()
	} else {
		token = p.nextToken()
	}
	if token == nil {
		p.unexpectedEOF(expected)
	}
	startElement, ok := token.(xml.StartElement)
	if !ok {
		p.unexpected(token, expected)
	}
	return startElement
}

func (p *xmlPlistParser) expectEndElement(skipSpace bool, expected xml.Token) xml.EndElement {
	var token xml.Token
	if skipSpace {
		token = p.nextNonSpace()
	} else {
		token = p.nextToken()
	}
	if token == nil {
		p.unexpectedEOF(expected)
	}
	endElement, ok := token.(xml.EndElement)
	if !ok {
		p.unexpected(token, expected)
	}
	return endElement
}

func (p *xmlPlistParser) expectCharData() string {
	expected := xml.CharData{}
	token := p.nextToken()
	if token == nil {
		p.unexpectedEOF(expected)
	}
	charData, ok := token.(xml.CharData)
	if !ok {
		p.unexpected(token, expected)
	}
	return string(charData.Copy())
}

func (p *xmlPlistParser) parseString(elementName string) string {
	charData := p.expectCharData()
	_ = p.expectEndElement(false, makeEndElement(elementName))
	return charData
}

func (p *xmlPlistParser) parseReal() float64 {
	charData := p.expectCharData()
	line, col := p.lineColer.LineCol(p.offset)
	f, err := strconv.ParseFloat(charData, 64)
	if err != nil {
		panic(errors.FileLineCol(
			p.filepath,
			line,
			col,
			fmt.Sprintf("%v", err),
		))
	}
	_ = p.expectEndElement(false, makeEndElement("real"))
	return f
}

func (p *xmlPlistParser) parseInteger() int64 {
	charData := p.expectCharData()
	line, col := p.lineColer.LineCol(p.offset)
	i, err := strconv.ParseInt(charData, 10, 64)
	if err != nil {
		panic(errors.FileLineCol(
			p.filepath,
			line,
			col,
			fmt.Sprintf("%v", err),
		))
	}
	_ = p.expectEndElement(false, makeEndElement("integer"))
	return i
}

func (p *xmlPlistParser) parseTrue() bool {
	_ = p.expectEndElement(false, makeEndElement("true"))
	return true
}

func (p *xmlPlistParser) parseFalse() bool {
	_ = p.expectEndElement(false, makeEndElement("false"))
	return false
}

func (p *xmlPlistParser) parseDate() time.Time {
	charData := p.expectCharData()
	line, col := p.lineColer.LineCol(p.offset)
	d, err := time.ParseInLocation(time.RFC3339, charData, time.UTC)
	if err != nil {
		panic(errors.FileLineCol(
			p.filepath,
			line,
			col,
			fmt.Sprintf("%v", err),
		))
	}
	_ = p.expectEndElement(false, makeEndElement("date"))
	return d
}

func (p *xmlPlistParser) parseData(line, col int) []byte {
	buf := bytes.Buffer{}
	for {
		token := p.nextToken()
		switch v := token.(type) {
		case xml.EndElement:
			if v.Name.Local == "data" {
				src := bytes.Trim(buf.Bytes(), "\r\n\t ")
				dst := make([]byte, base64.StdEncoding.DecodedLen(len(src)))
				_, err := base64.StdEncoding.Decode(dst, src)
				if err != nil {
					panic(errors.FileLineCol(
						p.filepath,
						line,
						col,
						fmt.Sprintf("%v", err),
					))
				}
				return dst
			}
			p.unexpected(token, makeEndElement("data"))
		case xml.CharData:
			buf.Write([]byte(v))
		default:
			p.unexpected(token, "CharData or </data>")
		}
	}
}

func (p *xmlPlistParser) parseArray() (out []interface{}) {
	for {
		token := p.nextNonSpace()
		if token == nil {
			p.unexpectedEOF(anyPlistValue)
		}
		switch v := token.(type) {
		case xml.EndElement:
			if v.Name.Local == "array" {
				return out
			}
			p.unexpected(token, makeEndElement("array"))
		case xml.StartElement:
			value := p.parseValue(v)
			out = append(out, value)
		default:
			p.unexpected(token, anyPlistValue)
		}
	}
}

func (p *xmlPlistParser) parseDict() map[string]interface{} {
	out := make(map[string]interface{})
	for {
		token := p.nextNonSpace()
		if token == nil {
			p.unexpectedEOF(anyPlistValue)
		}
		switch v := token.(type) {
		case xml.EndElement:
			if v.Name.Local == "dict" {
				return out
			}
			p.unexpected(token, makeEndElement("dict"))
		case xml.StartElement:
			if v.Name.Local != "key" {
				p.unexpected(token, makeStartElement("key"))
			}
			line, col := p.lineColer.LineCol(p.offset)
			key := p.parseString("key")
			startElement := p.expectStartElement(true, anyPlistValue)
			value := p.parseValue(startElement)
			if _, ok := out[key]; ok {
				panic(errors.FileLineCol(
					p.filepath,
					line,
					col,
					fmt.Sprintf("duplicated key `%v`", key),
				))
			}
			out[key] = value
		default:
			p.unexpected(token, anyPlistValue)
		}
	}
}

func (p *xmlPlistParser) parseValue(startElement xml.StartElement) Value {
	line, col := p.lineColer.LineCol(p.offset)
	switch startElement.Name.Local {
	case "string":
		s := p.parseString("string")
		return makeXMLPlistValue(s, line, col)
	case "real":
		f := p.parseReal()
		return makeXMLPlistValue(f, line, col)
	case "integer":
		i := p.parseInteger()
		return makeXMLPlistValue(i, line, col)
	case "true":
		b := p.parseTrue()
		return makeXMLPlistValue(b, line, col)
	case "false":
		b := p.parseFalse()
		return makeXMLPlistValue(b, line, col)
	case "date":
		t := p.parseDate()
		return makeXMLPlistValue(t, line, col)
	case "data":
		data := p.parseData(line, col)
		return makeXMLPlistValue(data, line, col)
	case "array":
		s := p.parseArray()
		return makeXMLPlistValue(s, line, col)
	case "dict":
		dict := p.parseDict()
		return makeXMLPlistValue(dict, line, col)
	default:
		p.unexpected(startElement, anyPlistValue)
	}
	return Value{}
}

func (p *xmlPlistParser) parsePlist() Value {
	_ = p.expectStartElement(true, makeStartElement("plist"))
	startElement := p.expectStartElement(true, anyPlistValue)
	out := p.parseValue(startElement)
	_ = p.expectEndElement(true, makeEndElement("plist"))
	return out
}

func (p *xmlPlistParser) expectEOF() {
	token := p.nextNonSpace()
	if token != nil {
		p.unexpected(token, nil)
	}
}

func (p *xmlPlistParser) parse() Value {
	p.expectXMLHeader()
	p.expectDocType()
	out := p.parsePlist()
	p.expectEOF()
	return out
}

// ParseXMLPlist parses XML plist.
func ParseXMLPlist(src, filepath string) (out Value, err error) {
	reader := strings.NewReader(src)
	decoder := xml.NewDecoder(reader)
	p := xmlPlistParser{
		decoder:   decoder,
		filepath:  filepath,
		lineColer: linecol.NewLineColer(src),
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
