package main

import (
	"os"
	"path"
	"path/filepath"
	"regexp"

	"github.com/iawaknahc/gogenstrings/errors"
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
	inEntries   map[string]entries
	inEntryMap  map[string]entryMap
	outEntryMap map[string]entryMap

	// Invocation of routine found in source code
	// The key is translation key
	routineCalls     routineCallSlice
	routineCallByKey map[string]routineCall
}

func newGenstringsContext(rootPath, devlang, routineName string, exclude *regexp.Regexp) genstringsContext {
	ctx := genstringsContext{
		rootPath:      rootPath,
		routineName:   routineName,
		devlang:       devlang,
		excludeRegexp: exclude,

		inEntries:   make(map[string]entries),
		inEntryMap:  make(map[string]entryMap),
		outEntryMap: make(map[string]entryMap),

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
	if err := p.readLocalizableDotStrings(); err != nil {
		return err
	}
	return p.readRoutineCalls()
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

func (p *genstringsContext) validate() error {
	if err := p.validateLocalizableDotStrings(); err != nil {
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
