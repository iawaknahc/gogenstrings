package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
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
				} else {
					escaping = false
				}
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

func getComment(c string) string {
	return strings.TrimSpace(c[2 : len(c)-2])
}

func getStringValue(s string) string {
	return s[1 : len(s)-1]
}

type entry struct {
	comment string
	key     string
	value   string
}

func (ls entry) mergeCall(rc routineCall) entry {
	ls.comment = rc.comment
	if ls.comment == "" {
		ls.comment = "No comment provided by engineer."
	}
	return ls
}

func (ls entry) mergeDev(dev entry) entry {
	ls.comment = dev.comment
	if ls.value == ls.key {
		ls.value = dev.value
	}
	return ls
}

func (ls entry) print() string {
	return "/* " + ls.comment + " */\n\"" + ls.key + `" = "` + ls.value + "\";\n\n"
}

type routineCall struct {
	startLine int
	startCol  int
	key       string
	comment   string
	// the first path this routine call is found
	path string
}

type entryMap map[string]entry

func (lss entryMap) mergeCalls(rcs map[string]routineCall) entryMap {
	output := entryMap{}
	// Copy existing LocalizableString if they are still in use
	for key, ls := range lss {
		if rc, ok := rcs[key]; ok {
			output[key] = ls.mergeCall(rc)
		}
	}
	// Copy new routine call
	for key, rc := range rcs {
		if _, ok := output[key]; !ok {
			output[key] = entry{
				comment: rc.comment,
				key:     rc.key,
				value:   rc.key,
			}
		}
	}
	return output
}

func (lss entryMap) merge(dev entryMap) entryMap {
	output := entryMap{}
	for key, ls := range lss {
		if devLs, ok := dev[key]; ok {
			output[key] = ls.mergeDev(devLs)
		}
	}
	for key, devLs := range dev {
		if _, ok := output[key]; !ok {
			output[key] = devLs
		}
	}
	return output
}

func (lss entryMap) sort() []entry {
	slice := []entry{}
	for _, ls := range lss {
		slice = append(slice, ls)
	}
	less := func(i, j int) bool {
		return slice[i].key < slice[j].key
	}
	sort.SliceStable(slice, less)
	return slice
}

type genstringsContext struct {
	rootPath        string
	lprojs          []string
	sourceFilePaths []string
	entryMapByLproj map[string]entryMap
	routineCalls    map[string]routineCall
	devlang         string
	routineName     string
	excludeRegexp   *regexp.Regexp
	mergedByLproj   map[string]entryMap
}

func newGenstringsContext(rootPath, developmentLanguage, routineName string, exclude *regexp.Regexp) genstringsContext {
	ctx := genstringsContext{
		rootPath:        rootPath,
		entryMapByLproj: make(map[string]entryMap),
		routineCalls:    make(map[string]routineCall),
		devlang:         developmentLanguage,
		routineName:     routineName,
		excludeRegexp:   exclude,
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
			p.entryMapByLproj[lproj] = entryMap{}
		} else {
			content, err := readFile(fullpath)
			if err != nil {
				return err
			}
			lss, err := parseStrings(content)
			if err != nil {
				return fmt.Errorf("%v in %v", err, fullpath)
			}
			p.entryMapByLproj[lproj] = lss
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
	p.mergedByLproj = make(map[string]entryMap)

	devLproj := p.devLproj()
	if devLproj == "" {
		return fmt.Errorf("cannot lproj of %v", p.devlang)
	}
	// Merge development language first
	existingLss, ok := p.entryMapByLproj[devLproj]
	if !ok {
		return fmt.Errorf("cannot find %v", devLproj)
	}
	p.mergedByLproj[devLproj] = existingLss.mergeCalls(p.routineCalls)

	// Merge other languages
	for lproj, lss := range p.entryMapByLproj {
		if lproj == devLproj {
			continue
		}
		p.mergedByLproj[lproj] = lss.merge(p.mergedByLproj[devLproj])
	}

	return nil
}

func (p *genstringsContext) write() error {
	for lproj, lss := range p.mergedByLproj {
		sorted := lss.sort()
		content := printStrings(sorted)
		targetPath := lproj + "/Localizable.strings"
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
	if err := p.write(); err != nil {
		return err
	}
	return nil
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

func printStrings(lss []entry) string {
	buf := bytes.Buffer{}
	for _, ls := range lss {
		buf.WriteString(ls.print())
	}
	return buf.String()
}
