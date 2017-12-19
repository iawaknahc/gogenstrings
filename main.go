package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	rootPtr := flag.String("root", ".", "the root path to the target")
	devLangPtr := flag.String("devlang", "en", "the development language")
	routinePtr := flag.String("routine", "NSLocalizedString", "the routine name to extract")
	flag.Parse()
	ctx := NewGenstringsContext(*rootPtr, *devLangPtr, *routinePtr)
	if err := ctx.Genstrings(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
