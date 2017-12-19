package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
)

func parseExclude(pattern string) (*regexp.Regexp, error) {
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
	excludeRe, err := parseExclude(*excludePtr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	ctx := NewGenstringsContext(*rootPtr, *devLangPtr, *routinePtr, excludeRe)
	if err := ctx.Genstrings(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
