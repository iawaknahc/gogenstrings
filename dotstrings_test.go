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
a = "b";
"b" = a;
	`
	expected := entries{
		entry{
			startLine: 3,
			startCol:  1,
			key:       "key_with_comment",
			value:     "value",
			comment:   " has_comment ",
		},
		entry{
			startLine: 4,
			startCol:  1,
			key:       "key_without_comment",
			value:     "value",
		},
		entry{
			startLine: 5,
			startCol:  1,
			key:       "a",
			value:     "b",
		},
		entry{
			startLine: 6,
			startCol:  1,
			key:       "b",
			value:     "a",
		},
	}
	actual, err := parseDotStrings(input, "")
	if err != nil {
		t.Fail()
	} else if !reflect.DeepEqual(actual, expected) {
		t.Fail()
	}
}

func TestParseDotStringsInvalid(t *testing.T) {
	cases := []struct {
		input string
		msg   string
	}{
		{"<>", ":1:1: not in .strings format"},
		{"()", ":1:1: not in .strings format"},
		{`a = <>;`, ":1:5: unexpected token"},
		{`a = ();`, ":1:5: unexpected token"},
	}
	for _, c := range cases {
		_, err := parseDotStrings(c.input, "")
		if err == nil {
			t.Fail()
		} else {
			msg := err.Error()
			if msg != c.msg {
				t.Fail()
			}
		}
	}
}
