package main

import (
	"reflect"
	"testing"
)

func TestParseDotStrings(t *testing.T) {
	input := `
/* has_comment */
"key_with_comment"= "value";

"key_without_comment" ="value";

	`
	expected := entries{
		entry{
			startLine: 3,
			startCol:  1,
			key:       "key_with_comment",
			value:     "value",
			comment:   "has_comment",
		},
		entry{
			startLine: 5,
			startCol:  1,
			key:       "key_without_comment",
			value:     "value",
		},
	}
	actual, err := parseDotStrings(input, "")
	if err != nil || !reflect.DeepEqual(actual, expected) {
		t.Fail()
	}

	input = `a = "b";`
	actual, err = parseDotStrings(input, "")
	if err == nil {
		t.Fail()
	}
}
