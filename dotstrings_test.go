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
	expected := entryMap{
		"key_with_comment": entry{
			key:     "key_with_comment",
			value:   "value",
			comment: "has_comment",
		},
		"key_without_comment": entry{
			key:   "key_without_comment",
			value: "value",
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

	input = `"a"="b";"a"="c";`
	actual, err = parseDotStrings(input, "")
	if err == nil {
		t.Fail()
	}
}
