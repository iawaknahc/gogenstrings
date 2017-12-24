package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"regexp"
)

func parseOptionalRegexp(pattern string) (*regexp.Regexp, error) {
	if pattern == "" {
		return nil, nil
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	return re, nil
}

func main() {
	rootPtr := flag.String("root", ".", "the root path to the target")
	infoPlistPtr := flag.String("infoplist", "<root>/Info.plist", "the path to Info.plist")
	devLangPtr := flag.String("devlang", "en", "the development language")
	routinePtr := flag.String("routine", "NSLocalizedString", "the routine name to extract")
	excludePtr := flag.String("exclude", "", "the regexp to exclude")
	flag.Parse()

	excludeRe, err := parseOptionalRegexp(*excludePtr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	rootPath := *rootPtr
	infoPlistPath := *infoPlistPtr
	if infoPlistPath == "" {
		infoPlistPath = path.Join(rootPath, "Info.plist")
	}
	devlang := *devLangPtr
	routineName := *routinePtr

	ctx := newGenstringsContext(
		rootPath,
		infoPlistPath,
		devlang,
		routineName,
		excludeRe,
	)
	if err := ctx.genstrings(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
