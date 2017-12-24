package main

import (
	"testing"
)

func TestFoo(t *testing.T) {
	ctx := newGenstringsContext(
		"./example",
		"./example/Info.plist",
		"en",
		"NSLocalizedString",
		nil,
	)
	if err := ctx.genstrings(); err != nil {
		t.Errorf("%v\n", err)
	}
}
