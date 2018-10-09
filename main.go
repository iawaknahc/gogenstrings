package main

import (
	"flag"
	"fmt"
	"os"
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
	devlang := *devLangPtr
	routineName := *routinePtr

	ctx := newGenstringsContext(
		rootPath,
		devlang,
		routineName,
		excludeRe,
	)
	if err := ctx.genstrings(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
