package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/iawaknahc/gogenstrings/errors"
	"github.com/iawaknahc/gogenstrings/xmlplist"
)

type infoPlist map[string]string

func xmlPlistValueToInfoPlist(v xmlplist.Value, filepath string) (infoPlist, error) {
	if _, ok := v.Value.(map[string]interface{}); !ok {
		return nil, errors.FileLineCol(
			filepath,
			v.Line,
			v.Col,
			fmt.Sprintf("unexpected %v; expected <dict>", v),
		)
	}
	dict := v.Flatten().(map[string]interface{})
	out := infoPlist{}
	for key, value := range dict {
		if s, ok := value.(string); ok {
			if isKeyValueLocalizable(key, s) {
				out[key] = s
			}
		}
	}
	return out, nil
}

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

type genstringsContext struct {
	// Configuration
	rootPath      string
	infoPlistPath string
	routineName   string
	devlang       string
	excludeRegexp *regexp.Regexp

	// Result of find
	lprojs          []string
	sourceFilePaths []string
	devLproj        string

	// Info.plist
	infoPlist infoPlist

	// Localizable.strings
	// The key is lproj
	inEntries   map[string]entries
	inEntryMap  map[string]entryMap
	outEntryMap map[string]entryMap

	// InfoPlist.strings
	// The key is lproj
	inInfoPlistEntries   map[string]entries
	inInfoPlistEntryMap  map[string]entryMap
	outInfoPlistEntryMap map[string]entryMap

	// Invocation of routine found in source code
	// The key is translation key
	routineCalls     routineCallSlice
	routineCallByKey map[string]routineCall
}

func newGenstringsContext(rootPath, infoPlistPath, devlang, routineName string, exclude *regexp.Regexp) genstringsContext {
	ctx := genstringsContext{
		rootPath:      rootPath,
		infoPlistPath: infoPlistPath,
		routineName:   routineName,
		devlang:       devlang,
		excludeRegexp: exclude,

		inEntries:   make(map[string]entries),
		inEntryMap:  make(map[string]entryMap),
		outEntryMap: make(map[string]entryMap),

		inInfoPlistEntries:   make(map[string]entries),
		inInfoPlistEntryMap:  make(map[string]entryMap),
		outInfoPlistEntryMap: make(map[string]entryMap),

		routineCalls:     []routineCall{},
		routineCallByKey: make(map[string]routineCall),
	}
	return ctx
}

func (p *genstringsContext) find() error {
	if err := p.findLprojs(); err != nil {
		return err
	}
	return p.findSourceFiles()
}

func (p *genstringsContext) findLprojs() error {
	lprojs, err := findLprojs(p.rootPath)
	if err != nil {
		return err
	}
	p.lprojs = lprojs

	targetBasename := p.devlang + ".lproj"

	for _, lproj := range p.lprojs {
		basename := filepath.Base(lproj)
		if basename == targetBasename {
			p.devLproj = lproj
			return nil
		}
	}
	return errors.File(path.Join(p.rootPath, targetBasename), "directory not found")
}

func (p *genstringsContext) findSourceFiles() error {
	sourceFilePaths, err := findSourceFiles(p.rootPath, p.excludeRegexp)
	if err != nil {
		return err
	}
	p.sourceFilePaths = sourceFilePaths
	return nil
}

func (p *genstringsContext) read() error {
	if err := p.readInfoPlist(); err != nil {
		return err
	}
	if err := p.readLocalizableDotStrings(); err != nil {
		return err
	}
	if err := p.readInfoPlistDotStrings(); err != nil {
		return err
	}
	return p.readRoutineCalls()
}

func (p *genstringsContext) readInfoPlist() error {
	content, err := readFile(p.infoPlistPath)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.File(p.infoPlistPath, "file not found")
		}
		return err
	}
	xmlPlistValue, err := xmlplist.ParseXMLPlist(content, p.infoPlistPath)
	if err != nil {
		return err
	}
	out, err := xmlPlistValueToInfoPlist(xmlPlistValue, p.infoPlistPath)
	if err != nil {
		return err
	}
	p.infoPlist = out
	return nil
}

func (p *genstringsContext) readLocalizableDotStrings() error {
	for _, lproj := range p.lprojs {
		fullpath := lproj + "/Localizable.strings"
		content, err := readFile(fullpath)
		if err != nil {
			if !os.IsNotExist(err) {
				return err
			}
			p.inEntries[lproj] = entries{}
		} else {
			es, err := parseDotStrings(content, fullpath)
			if err != nil {
				return err
			}
			p.inEntries[lproj] = es
		}
	}
	return nil
}

func (p *genstringsContext) readInfoPlistDotStrings() error {
	for _, lproj := range p.lprojs {
		fullpath := lproj + "/InfoPlist.strings"
		content, err := readFile(fullpath)
		if err != nil {
			if !os.IsNotExist(err) {
				return err
			}
			p.inInfoPlistEntries[lproj] = entries{}
		} else {
			es, err := parseDotStrings(content, fullpath)
			if err != nil {
				return err
			}
			p.inInfoPlistEntries[lproj] = es
		}
	}
	return nil
}

func (p *genstringsContext) validate() error {
	if err := p.validateLocalizableDotStrings(); err != nil {
		return err
	}
	if err := p.validateInfoPlistDotStrings(); err != nil {
		return err
	}
	return p.validateRoutineCalls()
}

func (p *genstringsContext) validateLocalizableDotStrings() error {
	for lproj, es := range p.inEntries {
		em, err := es.toEntryMap()
		if err != nil {
			return err
		}
		p.inEntryMap[lproj] = em
	}
	return nil
}

func (p *genstringsContext) validateInfoPlistDotStrings() error {
	for lproj, es := range p.inInfoPlistEntries {
		em, err := es.toEntryMap()
		if err != nil {
			return err
		}
		p.inInfoPlistEntryMap[lproj] = em
	}
	return nil
}

func (p *genstringsContext) validateRoutineCalls() error {
	out, err := p.routineCalls.toMap()
	if err != nil {
		return err
	}
	p.routineCallByKey = out
	return nil
}

func (p *genstringsContext) readRoutineCalls() error {
	for _, fullpath := range p.sourceFilePaths {
		content, err := readFile(fullpath)
		if err != nil {
			return err
		}
		calls, err := parseRoutineCalls(content, p.routineName, fullpath)
		if err != nil {
			return err
		}
		for _, call := range calls {
			p.routineCalls = append(p.routineCalls, call)
		}
	}
	return nil
}

func (p *genstringsContext) process() {
	devLproj := p.devLproj
	// Merge development language first
	oldDevEntryMap := p.inEntryMap[devLproj]
	p.outEntryMap[devLproj] = oldDevEntryMap.mergeCalls(p.routineCallByKey)

	// Merge other languages
	for lproj, em := range p.inEntryMap {
		if lproj == devLproj {
			continue
		}
		p.outEntryMap[lproj] = em.mergeDev(p.outEntryMap[devLproj])
	}

	// Merge development language first
	newInfoPlistEntryMap := p.infoPlist.toEntryMap()
	devInfoPlist := p.inInfoPlistEntryMap[devLproj].mergeDev(newInfoPlistEntryMap)
	p.outInfoPlistEntryMap[devLproj] = devInfoPlist

	// Merge InfoPlist.strings
	for lproj, em := range p.inInfoPlistEntryMap {
		if lproj == devLproj {
			continue
		}
		p.outInfoPlistEntryMap[lproj] = em.mergeDev(devInfoPlist)
	}
}

func (p *genstringsContext) write() error {
	// Write Localizable.strings
	for lproj, em := range p.outEntryMap {
		sorted := em.toEntries().sort()
		content := sorted.print(false)
		targetPath := lproj + "/Localizable.strings"
		if err := writeFile(targetPath, content); err != nil {
			return err
		}
	}
	// Write InfoPlist.strings
	for lproj, em := range p.outInfoPlistEntryMap {
		sorted := em.toEntries().sort()
		targetPath := lproj + "/InfoPlist.strings"
		if len(sorted) <= 0 {
			if err := os.Remove(targetPath); err != nil {
				if !os.IsNotExist(err) {
					return err
				}
			}
		} else {
			content := sorted.print(true)
			if err := writeFile(targetPath, content); err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *genstringsContext) genstrings() error {
	if err := p.find(); err != nil {
		return err
	}
	if err := p.read(); err != nil {
		return err
	}
	if err := p.validate(); err != nil {
		return err
	}
	p.process()
	return p.write()
}
