package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func lexComment(state stateFn) stateFn {
	return func(l *lexer) stateFn {
		l.next()
		l.next()
		for {
			if strings.HasPrefix(l.input[l.pos:], "*/") {
				l.next()
				l.next()
				l.emit(itemComment)
				return state
			}
			if r := l.next(); r == eof {
				return l.unexpectedToken(r)
			}
		}
	}
}

func lexSpaces(state stateFn) stateFn {
	return func(l *lexer) stateFn {
		for {
			r := l.next()
			if !isSpace(r) {
				if r != eof {
					l.backup()
				}
				if l.start < l.pos {
					l.emit(itemSpaces)
				}
				return state
			}
		}
	}
}

func lexString(state stateFn) stateFn {
	return func(l *lexer) stateFn {
		l.next()
		escaping := false
		for {
			r := l.next()
			switch r {
			case eof:
				return l.unexpectedToken(r)
			case '\\':
				escaping = !escaping
			case '"':
				if !escaping {
					l.emit(itemString)
					return state
				}
				escaping = false
			default:
				escaping = false
			}
		}
	}
}

func lexEntry(l *lexer) stateFn {
	for {
		if strings.HasPrefix(l.input[l.pos:], "/*") {
			return lexComment(lexEntry)
		}
		r := l.next()
		switch r {
		case eof:
			return l.eof()
		case '"':
			l.backup()
			return lexString(lexEntry)
		case ';':
			l.emit(itemSemicolon)
		case '=':
			l.emit(itemEqualSign)
		default:
			if isSpace(r) {
				l.backup()
				return lexSpaces(lexEntry)
			}
			return l.unexpectedToken(r)
		}
	}
}

func lexIdentifier(state stateFn) stateFn {
	return func(l *lexer) stateFn {
		for {
			r := l.next()
			if !isIdentifier(r) {
				if r != eof {
					l.backup()
				}
				l.emit(itemIdentifier)
				return state
			}
		}
	}
}

func lexRoutineCall(l *lexer) stateFn {
	for {
		r := l.next()
		switch r {
		case eof:
			return l.eof()
		case '"':
			l.backup()
			return lexString(lexRoutineCall)
		case '@':
			l.emit(itemAtSign)
		case '(':
			l.emit(itemParenLeft)
		case ')':
			l.emit(itemParenRight)
		case ':':
			l.emit(itemColon)
		case ',':
			l.emit(itemComma)
		default:
			if isSpace(r) {
				l.backup()
				return lexSpaces(lexRoutineCall)
			} else if isIdentifierStart(r) {
				return lexIdentifier(lexRoutineCall)
			} else {
				l.ignore()
			}
		}
	}
}

type genstringsContext struct {
	// Configuration
	rootPath      string
	routineName   string
	devlang       string
	excludeRegexp *regexp.Regexp

	// Result of find
	lprojs          []string
	sourceFilePaths []string

	// Localizable.strings
	// The key is lproj
	inStrings  map[string]entryMap
	outStrings map[string]entryMap

	// InfoPlist.strings
	// The key is lproj
	inInfoPlists  map[string]entryMap
	outInfoPlists map[string]entryMap

	// Invocation of routine found in source code
	// The key is translation key
	routineCalls map[string]routineCall
}

func newGenstringsContext(rootPath, developmentLanguage, routineName string, exclude *regexp.Regexp) genstringsContext {
	ctx := genstringsContext{
		rootPath:      rootPath,
		routineName:   routineName,
		devlang:       developmentLanguage,
		excludeRegexp: exclude,
		inStrings:     make(map[string]entryMap),
		outStrings:    make(map[string]entryMap),
		inInfoPlists:  make(map[string]entryMap),
		outInfoPlists: make(map[string]entryMap),
		routineCalls:  make(map[string]routineCall),
	}
	return ctx
}

func (p *genstringsContext) readLprojs() error {
	lprojs, err := findLprojs(p.rootPath)
	if err != nil {
		return err
	}
	p.lprojs = lprojs

	for _, lproj := range p.lprojs {
		fullpath := lproj + "/Localizable.strings"
		_, err := os.Stat(fullpath)
		if err != nil {
			if !os.IsNotExist(err) {
				return err
			}
			p.inStrings[lproj] = entryMap{}
		} else {
			content, err := readFile(fullpath)
			if err != nil {
				return err
			}
			lss, err := parseStrings(content)
			if err != nil {
				return fmt.Errorf("%v in %v", err, fullpath)
			}
			p.inStrings[lproj] = lss
		}
	}

	for _, lproj := range p.lprojs {
		fullpath := lproj + "/InfoPlist.strings"
		_, err := os.Stat(fullpath)
		if err != nil {
			if !os.IsNotExist(err) {
				return err
			}
			p.inInfoPlists[lproj] = entryMap{}
		} else {
			content, err := readFile(fullpath)
			if err != nil {
				return err
			}
			lss, err := parseStrings(content)
			if err != nil {
				return fmt.Errorf("%v in %v", err, fullpath)
			}
			p.inInfoPlists[lproj] = lss
		}
	}

	return nil
}

func (p *genstringsContext) readRoutineCalls() error {
	sourceFilePaths, err := findSourceFiles(p.rootPath, p.excludeRegexp)
	if err != nil {
		return err
	}
	p.sourceFilePaths = sourceFilePaths
	for _, fullpath := range p.sourceFilePaths {
		content, err := readFile(fullpath)
		if err != nil {
			return err
		}
		calls, err := parseRoutineCall(content, p.routineName)
		if err != nil {
			return fmt.Errorf("%v in %v", err, fullpath)
		}
		for _, call := range calls {
			if call.key == "" {
				return fmt.Errorf(
					"routine call at %v:%v in %v has empty key",
					call.startLine,
					call.startCol,
					fullpath,
				)
			}
			existingCall, ok := p.routineCalls[call.key]
			if ok {
				if call.comment != existingCall.comment {
					return fmt.Errorf(
						"\nroutine call `%v` at %v:%v in %v\nroutine call `%v` at %v:%v in %v\nhave different comments",
						existingCall.key,
						existingCall.startLine,
						existingCall.startCol,
						existingCall.path,
						call.key,
						call.startLine,
						call.startCol,
						fullpath,
					)
				}
			} else {
				call.path = fullpath
				p.routineCalls[call.key] = call
			}
		}
	}
	return nil
}

func (p *genstringsContext) devLproj() string {
	for _, lproj := range p.lprojs {
		basename := filepath.Base(lproj)
		if basename == p.devlang+".lproj" {
			return lproj
		}
	}
	return ""
}

func (p *genstringsContext) merge() error {
	devLproj := p.devLproj()
	if devLproj == "" {
		return fmt.Errorf("cannot lproj of %v", p.devlang)
	}
	// Merge development language first
	existingLss, ok := p.inStrings[devLproj]
	if !ok {
		return fmt.Errorf("cannot find %v", devLproj)
	}
	p.outStrings[devLproj] = existingLss.mergeCalls(p.routineCalls)

	// Merge other languages
	for lproj, lss := range p.inStrings {
		if lproj == devLproj {
			continue
		}
		p.outStrings[lproj] = lss.mergeDev(p.outStrings[devLproj])
	}

	// Merge InfoPlist.strings
	devInfoPlist := p.inInfoPlists[devLproj]
	for lproj, lss := range p.inInfoPlists {
		if lproj == devLproj {
			p.outInfoPlists[lproj] = devInfoPlist
		} else {
			p.outInfoPlists[lproj] = lss.mergeDev(devInfoPlist)
		}
	}

	return nil
}

func (p *genstringsContext) write() error {
	// Write Localizable.strings
	for lproj, lss := range p.outStrings {
		sorted := lss.sort()
		content := printStrings(sorted, false)
		targetPath := lproj + "/Localizable.strings"
		if err := writeFile(targetPath, content); err != nil {
			return err
		}
	}
	// Write InfoPlist.strings
	for lproj, lss := range p.outInfoPlists {
		sorted := lss.sort()
		if len(sorted) <= 0 {
			continue
		}
		content := printStrings(sorted, true)
		targetPath := lproj + "/InfoPlist.strings"
		if err := writeFile(targetPath, content); err != nil {
			return err
		}
	}

	return nil
}

func (p *genstringsContext) genstrings() error {
	if err := p.readLprojs(); err != nil {
		return err
	}
	if err := p.readRoutineCalls(); err != nil {
		return err
	}
	if err := p.merge(); err != nil {
		return err
	}
	return p.write()
}

type stringsParser struct {
	lexer *lexer
}

func (p *stringsParser) parse() (output []entry, err error) {
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
			comment: comment,
			key:     getStringValue(key.Value),
			value:   getStringValue(value.Value),
		}
		output = append(output, ls)
	}
	return output, nil
}

func (p *stringsParser) nextNonSpace() lexItem {
	for {
		item := p.lexer.nextItem()
		if item.Type != itemSpaces {
			return item
		}
	}
}

func (p *stringsParser) recover(errp *error) {
	if r := recover(); r != nil {
		err, ok := r.(error)
		if !ok {
			panic("panicked without error")
		}
		*errp = err
	}
}

func (p *stringsParser) expect(expected itemType) lexItem {
	item := p.nextNonSpace()
	if item.Type != expected {
		p.unexpected(item)
	}
	return item
}

func (p *stringsParser) unexpected(item lexItem) {
	if item.Type == itemError {
		panic(item.Err)
	} else {
		panic(fmt.Errorf("unexpected token %v", item))
	}
}

func parseStrings(src string) (entryMap, error) {
	l := newLexer(src, lexEntry)
	p := &stringsParser{
		lexer: &l,
	}
	lss, err := p.parse()
	if err != nil {
		return nil, err
	}
	output := entryMap{}
	for _, ls := range lss {
		if _, ok := output[ls.key]; ok {
			return nil, fmt.Errorf("duplicated key %q", ls.key)
		}
		output[ls.key] = ls
	}
	return output, nil
}

func printStrings(lss []entry, suppressEmptyComment bool) string {
	buf := bytes.Buffer{}
	for _, ls := range lss {
		buf.WriteString(ls.print(suppressEmptyComment))
	}
	return buf.String()
}
