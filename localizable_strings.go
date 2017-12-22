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

func lexComment(state StateFn) StateFn {
	return func(l *Lexer) StateFn {
		l.Next()
		l.Next()
		for {
			if strings.HasPrefix(l.input[l.pos:], "*/") {
				l.Next()
				l.Next()
				l.Emit(ItemComment)
				return state
			}
			if r := l.Next(); r == EOF {
				return l.UnexpectedToken(r)
			}
		}
	}
}

func lexSpaces(state StateFn) StateFn {
	return func(l *Lexer) StateFn {
		for {
			r := l.Next()
			if !isSpace(r) {
				if r != EOF {
					l.Backup()
				}
				if l.start < l.pos {
					l.Emit(ItemSpaces)
				}
				return state
			}
		}
	}
}

func lexString(state StateFn) StateFn {
	return func(l *Lexer) StateFn {
		l.Next()
		escaping := false
		for {
			r := l.Next()
			switch r {
			case EOF:
				return l.UnexpectedToken(r)
			case '\\':
				escaping = !escaping
			case '"':
				if !escaping {
					l.Emit(ItemString)
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

func lexLocalizableString(l *Lexer) StateFn {
	for {
		if strings.HasPrefix(l.input[l.pos:], "/*") {
			return lexComment(lexLocalizableString)
		}
		r := l.Next()
		switch r {
		case EOF:
			return l.EOF()
		case '"':
			l.Backup()
			return lexString(lexLocalizableString)
		case ';':
			l.Emit(ItemSemicolon)
		case '=':
			l.Emit(ItemEqualSign)
		default:
			if isSpace(r) {
				l.Backup()
				return lexSpaces(lexLocalizableString)
			}
			return l.UnexpectedToken(r)
		}
	}
}

func lexIdentifier(state StateFn) StateFn {
	return func(l *Lexer) StateFn {
		for {
			r := l.Next()
			if !isIdentifier(r) {
				if r != EOF {
					l.Backup()
				}
				l.Emit(ItemIdentifier)
				return state
			}
		}
	}
}

func lexRoutineCall(l *Lexer) StateFn {
	for {
		r := l.Next()
		switch r {
		case EOF:
			return l.EOF()
		case '"':
			l.Backup()
			return lexString(lexRoutineCall)
		case '@':
			l.Emit(ItemAtSign)
		case '(':
			l.Emit(ItemParenLeft)
		case ')':
			l.Emit(ItemParenRight)
		case ':':
			l.Emit(ItemColon)
		case ',':
			l.Emit(ItemComma)
		default:
			if isSpace(r) {
				l.Backup()
				return lexSpaces(lexRoutineCall)
			} else if isIdentifierStart(r) {
				return lexIdentifier(lexRoutineCall)
			} else {
				l.Ignore()
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

func joinStringLiteral(s1, s2 string) string {
	v1 := getStringValue(s1)
	v2 := getStringValue(s2)
	return `"` + v1 + v2 + `"`
}

type LocalizableString struct {
	Comment string
	Key     string
	Value   string
}

func (ls LocalizableString) MergeCall(rc RoutineCall) LocalizableString {
	ls.Comment = rc.Comment
	if ls.Comment == "" {
		ls.Comment = "No comment provided by engineer."
	}
	return ls
}

func (ls LocalizableString) MergeDev(dev LocalizableString) LocalizableString {
	ls.Comment = dev.Comment
	if ls.Value == ls.Key {
		ls.Value = dev.Value
	}
	return ls
}

func (ls LocalizableString) Print() string {
	return "/* " + ls.Comment + " */\n\"" + ls.Key + `" = "` + ls.Value + "\";\n\n"
}

type RoutineCall struct {
	StartLine int
	StartCol  int
	Key       string
	Comment   string
	// the first path this routine call is found
	path string
}

type LocalizableStrings map[string]LocalizableString

func (lss LocalizableStrings) MergeCalls(rcs map[string]RoutineCall) LocalizableStrings {
	output := LocalizableStrings{}
	// Copy existing LocalizableString if they are still in use
	for key, ls := range lss {
		if rc, ok := rcs[key]; ok {
			output[key] = ls.MergeCall(rc)
		}
	}
	// Copy new routine call
	for key, rc := range rcs {
		if _, ok := output[key]; !ok {
			output[key] = LocalizableString{
				Comment: rc.Comment,
				Key:     rc.Key,
				Value:   rc.Key,
			}
		}
	}
	return output
}

func (lss LocalizableStrings) Merge(dev LocalizableStrings) LocalizableStrings {
	output := LocalizableStrings{}
	for key, ls := range lss {
		if devLs, ok := dev[key]; ok {
			output[key] = ls.MergeDev(devLs)
		}
	}
	for key, devLs := range dev {
		if _, ok := output[key]; !ok {
			output[key] = devLs
		}
	}
	return output
}

func (lss LocalizableStrings) Sort() []LocalizableString {
	slice := []LocalizableString{}
	for _, ls := range lss {
		slice = append(slice, ls)
	}
	less := func(i, j int) bool {
		return slice[i].Key < slice[j].Key
	}
	sort.SliceStable(slice, less)
	return slice
}

type GenstringsContext struct {
	RootPath                  string
	Lprojs                    []string
	SourceFilePaths           []string
	LocalizableStringsByLproj map[string]LocalizableStrings
	RoutineCalls              map[string]RoutineCall
	DevelopmentLanguage       string
	RoutineName               string
	ExcludeRegexp             *regexp.Regexp
	mergedByLproj             map[string]LocalizableStrings
}

func NewGenstringsContext(rootPath, developmentLanguage, routineName string, exclude *regexp.Regexp) GenstringsContext {
	ctx := GenstringsContext{
		RootPath:                  rootPath,
		LocalizableStringsByLproj: make(map[string]LocalizableStrings),
		RoutineCalls:              make(map[string]RoutineCall),
		DevelopmentLanguage:       developmentLanguage,
		RoutineName:               routineName,
		ExcludeRegexp:             exclude,
	}
	return ctx
}

func (p *GenstringsContext) ReadLprojs() error {
	lprojs, err := FindLprojs(p.RootPath)
	if err != nil {
		return err
	}
	p.Lprojs = lprojs
	for _, lproj := range p.Lprojs {
		fullpath := lproj + "/Localizable.strings"
		_, err := os.Stat(fullpath)
		if err != nil {
			if !os.IsNotExist(err) {
				return err
			}
			p.LocalizableStringsByLproj[lproj] = LocalizableStrings{}
		} else {
			content, err := ReadFile(fullpath)
			if err != nil {
				return err
			}
			lss, err := ParseLocalizableStrings(content)
			if err != nil {
				return fmt.Errorf("%v in %v", err, fullpath)
			}
			p.LocalizableStringsByLproj[lproj] = lss
		}
	}
	return nil
}

func (p *GenstringsContext) ReadRoutineCalls() error {
	sourceFilePaths, err := FindSourceCodeFiles(p.RootPath, p.ExcludeRegexp)
	if err != nil {
		return err
	}
	p.SourceFilePaths = sourceFilePaths
	for _, fullpath := range p.SourceFilePaths {
		content, err := ReadFile(fullpath)
		if err != nil {
			return err
		}
		calls, err := ParseRoutineCall(content, p.RoutineName)
		if err != nil {
			return fmt.Errorf("%v in %v", err, fullpath)
		}
		for _, call := range calls {
			if call.Key == "" {
				return fmt.Errorf(
					"routine call at %v:%v in %v has empty key",
					call.StartLine,
					call.StartCol,
					fullpath,
				)
			}
			existingCall, ok := p.RoutineCalls[call.Key]
			if ok {
				if call.Comment != existingCall.Comment {
					return fmt.Errorf(
						"\nroutine call `%v` at %v:%v in %v\nroutine call `%v` at %v:%v in %v\nhave different comments",
						existingCall.Key,
						existingCall.StartLine,
						existingCall.StartCol,
						existingCall.path,
						call.Key,
						call.StartLine,
						call.StartCol,
						fullpath,
					)
				}
			} else {
				call.path = fullpath
				p.RoutineCalls[call.Key] = call
			}
		}
	}
	return nil
}

func (p *GenstringsContext) DevLproj() string {
	for _, lproj := range p.Lprojs {
		basename := filepath.Base(lproj)
		if basename == p.DevelopmentLanguage+".lproj" {
			return lproj
		}
	}
	return ""
}

func (p *GenstringsContext) Merge() error {
	p.mergedByLproj = make(map[string]LocalizableStrings)

	devLproj := p.DevLproj()
	if devLproj == "" {
		return fmt.Errorf("cannot lproj of %v", p.DevelopmentLanguage)
	}
	// Merge development language first
	existingLss, ok := p.LocalizableStringsByLproj[devLproj]
	if !ok {
		return fmt.Errorf("cannot find %v", devLproj)
	}
	p.mergedByLproj[devLproj] = existingLss.MergeCalls(p.RoutineCalls)

	// Merge other languages
	for lproj, lss := range p.LocalizableStringsByLproj {
		if lproj == devLproj {
			continue
		}
		p.mergedByLproj[lproj] = lss.Merge(p.mergedByLproj[devLproj])
	}

	return nil
}

func (p *GenstringsContext) Write() error {
	for lproj, lss := range p.mergedByLproj {
		sorted := lss.Sort()
		content := PrintLocalizableStrings(sorted)
		targetPath := lproj + "/Localizable.strings"
		if err := WriteFile(targetPath, content); err != nil {
			return err
		}
	}
	return nil
}

func (p *GenstringsContext) Genstrings() error {
	if err := p.ReadLprojs(); err != nil {
		return err
	}
	if err := p.ReadRoutineCalls(); err != nil {
		return err
	}
	if err := p.Merge(); err != nil {
		return err
	}
	if err := p.Write(); err != nil {
		return err
	}
	return nil
}

type LocalizableStringsParser struct {
	lexer *Lexer
}

func (p *LocalizableStringsParser) parse() (output []LocalizableString, err error) {
	defer p.recover(&err)
	for {
		token := p.nextNonSpace()
		if token.Type == ItemEOF {
			break
		}
		var key Item
		var comment string

		if token.Type == ItemComment {
			comment = getComment(token.Value)
			key = p.expect(ItemString)
		} else if token.Type == ItemString {
			comment = ""
			key = token
		} else {
			p.unexpected(token)
		}

		p.expect(ItemEqualSign)
		value := p.expect(ItemString)
		p.expect(ItemSemicolon)
		ls := LocalizableString{
			Comment: comment,
			Key:     getStringValue(key.Value),
			Value:   getStringValue(value.Value),
		}
		output = append(output, ls)
	}
	return output, nil
}

func (p *LocalizableStringsParser) nextNonSpace() Item {
	for {
		item := p.lexer.NextItem()
		if item.Type != ItemSpaces {
			return item
		}
	}
}

func (p *LocalizableStringsParser) recover(errp *error) {
	if r := recover(); r != nil {
		err, ok := r.(error)
		if !ok {
			panic("panicked without error")
		}
		*errp = err
	}
}

func (p *LocalizableStringsParser) expect(expected ItemType) Item {
	item := p.nextNonSpace()
	if item.Type != expected {
		p.unexpected(item)
	}
	return item
}

func (p *LocalizableStringsParser) unexpected(item Item) {
	if item.Type == ItemError {
		panic(item.Err)
	} else {
		panic(fmt.Errorf("unexpected token %v", item))
	}
}

func ParseLocalizableStrings(src string) (LocalizableStrings, error) {
	l := NewLexer(src, lexLocalizableString)
	p := &LocalizableStringsParser{
		lexer: &l,
	}
	lss, err := p.parse()
	if err != nil {
		return nil, err
	}
	output := LocalizableStrings{}
	for _, ls := range lss {
		if _, ok := output[ls.Key]; ok {
			return nil, fmt.Errorf("duplicated key %q", ls.Key)
		}
		output[ls.Key] = ls
	}
	return output, nil
}

func ParseRoutineCall(src, routineName string) ([]RoutineCall, error) {
	l := NewLexer(src, lexRoutineCall)
	p := &RoutineCallParser{
		routineName: routineName,
		lexer:       &l,
	}
	return p.parse()
}

type RoutineCallParser struct {
	routineName string
	lexer       *Lexer
	peekCount   int
	token       [1]Item
}

func (p *RoutineCallParser) next() Item {
	if p.peekCount > 0 {
		p.peekCount--
	} else {
		p.token[0] = p.lexer.NextItem()
	}
	return p.token[p.peekCount]
}

func (p *RoutineCallParser) backup() {
	p.peekCount++
}

func (p *RoutineCallParser) nextNonSpace() (item Item) {
	for {
		item = p.next()
		if item.Type != ItemSpaces {
			break
		}
	}
	return item
}

func (p *RoutineCallParser) recover(errp *error) {
	if r := recover(); r != nil {
		err, ok := r.(error)
		if !ok {
			panic("panicked without error")
		}
		*errp = err
	}
}

func (p *RoutineCallParser) expect(expected ItemType) Item {
	item := p.nextNonSpace()
	if item.Type != expected {
		p.unexpected(item)
	}
	return item
}

func (p *RoutineCallParser) unexpected(item Item) {
	if item.Type == ItemError {
		panic(item.Err)
	} else {
		panic(fmt.Errorf("unexpected token %v", item))
	}
}

func (p *RoutineCallParser) parse() (output []RoutineCall, outerr error) {
	defer p.recover(&outerr)
	for {
		token := p.nextNonSpace()
		if token.Type == ItemEOF {
			break
		}
		if token.Type != ItemIdentifier || token.Value != p.routineName {
			continue
		}
		p.expect(ItemParenLeft)
		key := p.parseString()
		p.expect(ItemComma)
		p.parseFuncLabel()
		comment := p.parseString()
		p.expect(ItemParenRight)
		rc := RoutineCall{
			StartLine: token.StartLine,
			StartCol:  token.StartCol,
			Key:       key,
			Comment:   comment,
		}
		output = append(output, rc)
	}
	return output, nil
}

func (p *RoutineCallParser) parseString() (output string) {
	atSign := false
	token := p.nextNonSpace()

	if token.Type == ItemAtSign {
		atSign = true
		token = p.nextNonSpace()
		if token.Type != ItemString {
			p.unexpected(token)
		}
		output += getStringValue(token.Value)
	} else if token.Type == ItemString {
		output += getStringValue(token.Value)
	} else {
		p.unexpected(token)
	}

	for {
		token = p.nextNonSpace()
		if atSign && token.Type == ItemAtSign {
			token = p.nextNonSpace()
			if token.Type != ItemString {
				p.unexpected(token)
				break
			}
			output += getStringValue(token.Value)
		} else if !atSign && token.Type == ItemString {
			output += getStringValue(token.Value)
		} else {
			p.backup()
			break
		}
	}

	return output
}

func (p *RoutineCallParser) parseFuncLabel() {
	token := p.nextNonSpace()
	if token.Type != ItemIdentifier {
		p.backup()
		return
	}
	p.expect(ItemColon)
}

func FindLprojs(root string) (output []string, outerr error) {
	walkFn := func(fullpath string, info os.FileInfo, err error) error {
		if err != nil {
			outerr = err
			return err
		}
		if info.IsDir() {
			if strings.HasSuffix(fullpath, ".lproj") && !strings.HasSuffix(fullpath, "Base.lproj") {
				output = append(output, fullpath)
			}
		}
		return nil
	}
	filepath.Walk(root, walkFn)
	return
}

func isSourceCodeFile(fullpath string) bool {
	ext := filepath.Ext(fullpath)
	return ext == ".swift" || ext == ".m" || ext == ".h"
}

func FindSourceCodeFiles(root string, exclude *regexp.Regexp) (output []string, outerr error) {
	walkFn := func(fullpath string, info os.FileInfo, err error) error {
		if err != nil {
			outerr = err
			return err
		}
		if info.Mode().IsRegular() && isSourceCodeFile(fullpath) {
			if exclude == nil || !exclude.MatchString(fullpath) {
				output = append(output, fullpath)
			}
		}
		return nil
	}
	filepath.Walk(root, walkFn)
	return
}

func PrintLocalizableStrings(lss []LocalizableString) string {
	buf := bytes.Buffer{}
	for _, ls := range lss {
		buf.WriteString(ls.Print())
	}
	return buf.String()
}
