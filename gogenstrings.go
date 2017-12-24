package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

type genstringsContext struct {
	// Configuration
	rootPath      string
	routineName   string
	devlang       string
	excludeRegexp *regexp.Regexp

	// Result of find
	lprojs          []string
	sourceFilePaths []string
	devLproj        string

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
	routineCalls     []routineCall
	routineCallByKey map[string]routineCall
}

func newGenstringsContext(rootPath, developmentLanguage, routineName string, exclude *regexp.Regexp) genstringsContext {
	ctx := genstringsContext{
		rootPath:         rootPath,
		routineName:      routineName,
		devlang:          developmentLanguage,
		excludeRegexp:    exclude,
		inStrings:        make(map[string]entryMap),
		outStrings:       make(map[string]entryMap),
		inInfoPlists:     make(map[string]entryMap),
		outInfoPlists:    make(map[string]entryMap),
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

	for _, lproj := range p.lprojs {
		basename := filepath.Base(lproj)
		if basename == p.devlang+".lproj" {
			p.devLproj = lproj
			return nil
		}
	}
	return fmt.Errorf("cannot lproj of %v", p.devlang)
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
	if err := p.readLocalizableDotStrings(); err != nil {
		return err
	}
	if err := p.readInfoPlistDotStrings(); err != nil {
		return err
	}
	return p.readRoutineCalls()
}

func (p *genstringsContext) readLocalizableDotStrings() error {
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
			lss, err := parseDotStrings(content)
			if err != nil {
				return fmt.Errorf("%v in %v", err, fullpath)
			}
			p.inStrings[lproj] = lss
		}
	}
	return nil
}

func (p *genstringsContext) readInfoPlistDotStrings() error {
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
			lss, err := parseDotStrings(content)
			if err != nil {
				return fmt.Errorf("%v in %v", err, fullpath)
			}
			p.inInfoPlists[lproj] = lss
		}
	}
	return nil
}

func (p *genstringsContext) validate() error {
	return p.validateRoutineCalls()
}

func (p *genstringsContext) validateRoutineCalls() error {
	for _, call := range p.routineCalls {
		// Validate every call has non-empty key
		if call.key == "" {
			return fmt.Errorf(
				"routine call at %v:%v in %v has empty key",
				call.startLine,
				call.startCol,
				call.filepath,
			)
		}

		// Validate calls having the same key has the same comment
		existingCall, ok := p.routineCallByKey[call.key]
		if ok {
			if call.comment != existingCall.comment {
				return fmt.Errorf(
					"\nroutine call `%v` at %v:%v in %v\nroutine call `%v` at %v:%v in %v\nhave different comments",
					existingCall.key,
					existingCall.startLine,
					existingCall.startCol,
					existingCall.filepath,
					call.key,
					call.startLine,
					call.startCol,
					call.filepath,
				)
			}
		}
		p.routineCallByKey[call.key] = call
	}

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
			return fmt.Errorf("%v in %v", err, fullpath)
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
	oldDevEntryMap := p.inStrings[devLproj]
	p.outStrings[devLproj] = oldDevEntryMap.mergeCalls(p.routineCallByKey)

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
}

func (p *genstringsContext) write() error {
	// Write Localizable.strings
	for lproj, lss := range p.outStrings {
		sorted := lss.toEntries().sort()
		content := sorted.print(false)
		targetPath := lproj + "/Localizable.strings"
		if err := writeFile(targetPath, content); err != nil {
			return err
		}
	}
	// Write InfoPlist.strings
	for lproj, lss := range p.outInfoPlists {
		sorted := lss.toEntries().sort()
		if len(sorted) <= 0 {
			continue
		}
		content := sorted.print(true)
		targetPath := lproj + "/InfoPlist.strings"
		if err := writeFile(targetPath, content); err != nil {
			return err
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
