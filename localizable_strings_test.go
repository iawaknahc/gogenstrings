package main

import (
	"testing"
)

func TestFoo(t *testing.T) {
	ctx := NewGenstringsContext("./example", "en", "NSLocalizedString")
	if err := ctx.Genstrings(); err != nil {
		t.Errorf("%v\n", err)
	}
}
