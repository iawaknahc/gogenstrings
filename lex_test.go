package main

import (
	"reflect"
	"testing"
)

func drainLexer(l *lexer) (out []lexItem) {
	for {
		item := l.nextItem()
		out = append(out, item)
		if item.Type == itemError || item.Type == itemEOF {
			return out
		}
	}
}

func TestLexASCIIPlist(t *testing.T) {
	input := `
	{
		$-_.:/ = (1, 2);
		a = <dead beef>;
	}
`
	l := newLexer(input, "", lexASCIIPlist)
	actual := drainLexer(&l)
	expected := []lexItem{
		lexItem{
			Type:      itemSpaces,
			RawValue:  "\n\t",
			Value:     "\n\t",
			Start:     0,
			End:       2,
			StartLine: 2,
			StartCol:  0,
			EndLine:   2,
			EndCol:    2,
		},
		lexItem{
			Type:      itemBraceLeft,
			RawValue:  "{",
			Value:     "{",
			Start:     2,
			End:       3,
			StartLine: 2,
			StartCol:  2,
			EndLine:   3,
			EndCol:    0,
		},
		lexItem{
			Type:      itemSpaces,
			RawValue:  "\n\t\t",
			Value:     "\n\t\t",
			Start:     3,
			End:       6,
			StartLine: 3,
			StartCol:  0,
			EndLine:   3,
			EndCol:    3,
		},
		lexItem{
			Type:      itemBareString,
			RawValue:  "$-_.:/",
			Value:     "$-_.:/",
			Start:     6,
			End:       12,
			StartLine: 3,
			StartCol:  3,
			EndLine:   3,
			EndCol:    9,
		},
		lexItem{
			Type:      itemSpaces,
			RawValue:  " ",
			Value:     " ",
			Start:     12,
			End:       13,
			StartLine: 3,
			StartCol:  9,
			EndLine:   3,
			EndCol:    10,
		},
		lexItem{
			Type:      itemEqualSign,
			RawValue:  "=",
			Value:     "=",
			Start:     13,
			End:       14,
			StartLine: 3,
			StartCol:  10,
			EndLine:   3,
			EndCol:    11,
		},
		lexItem{
			Type:      itemSpaces,
			RawValue:  " ",
			Value:     " ",
			Start:     14,
			End:       15,
			StartLine: 3,
			StartCol:  11,
			EndLine:   3,
			EndCol:    12,
		},
		lexItem{
			Type:      itemParenLeft,
			RawValue:  "(",
			Value:     "(",
			Start:     15,
			End:       16,
			StartLine: 3,
			StartCol:  12,
			EndLine:   3,
			EndCol:    13,
		},
		lexItem{
			Type:      itemBareString,
			RawValue:  "1",
			Value:     "1",
			Start:     16,
			End:       17,
			StartLine: 3,
			StartCol:  13,
			EndLine:   3,
			EndCol:    14,
		},
		lexItem{
			Type:      itemComma,
			RawValue:  ",",
			Value:     ",",
			Start:     17,
			End:       18,
			StartLine: 3,
			StartCol:  14,
			EndLine:   3,
			EndCol:    15,
		},
		lexItem{
			Type:      itemSpaces,
			RawValue:  " ",
			Value:     " ",
			Start:     18,
			End:       19,
			StartLine: 3,
			StartCol:  15,
			EndLine:   3,
			EndCol:    16,
		},
		lexItem{
			Type:      itemBareString,
			RawValue:  "2",
			Value:     "2",
			Start:     19,
			End:       20,
			StartLine: 3,
			StartCol:  16,
			EndLine:   3,
			EndCol:    17,
		},
		lexItem{
			Type:      itemParenRight,
			RawValue:  ")",
			Value:     ")",
			Start:     20,
			End:       21,
			StartLine: 3,
			StartCol:  17,
			EndLine:   3,
			EndCol:    18,
		},
		lexItem{
			Type:      itemSemicolon,
			RawValue:  ";",
			Value:     ";",
			Start:     21,
			End:       22,
			StartLine: 3,
			StartCol:  18,
			EndLine:   4,
			EndCol:    0,
		},
		lexItem{
			Type:      itemSpaces,
			RawValue:  "\n\t\t",
			Value:     "\n\t\t",
			Start:     22,
			End:       25,
			StartLine: 4,
			StartCol:  0,
			EndLine:   4,
			EndCol:    3,
		},
		lexItem{
			Type:      itemBareString,
			RawValue:  "a",
			Value:     "a",
			Start:     25,
			End:       26,
			StartLine: 4,
			StartCol:  3,
			EndLine:   4,
			EndCol:    4,
		},
		lexItem{
			Type:      itemSpaces,
			RawValue:  " ",
			Value:     " ",
			Start:     26,
			End:       27,
			StartLine: 4,
			StartCol:  4,
			EndLine:   4,
			EndCol:    5,
		},
		lexItem{
			Type:      itemEqualSign,
			RawValue:  "=",
			Value:     "=",
			Start:     27,
			End:       28,
			StartLine: 4,
			StartCol:  5,
			EndLine:   4,
			EndCol:    6,
		},
		lexItem{
			Type:      itemSpaces,
			RawValue:  " ",
			Value:     " ",
			Start:     28,
			End:       29,
			StartLine: 4,
			StartCol:  6,
			EndLine:   4,
			EndCol:    7,
		},
		lexItem{
			Type:      itemLessThanSign,
			RawValue:  "<",
			Value:     "<",
			Start:     29,
			End:       30,
			StartLine: 4,
			StartCol:  7,
			EndLine:   4,
			EndCol:    8,
		},
		lexItem{
			Type:      itemBareString,
			RawValue:  "dead",
			Value:     "dead",
			Start:     30,
			End:       34,
			StartLine: 4,
			StartCol:  8,
			EndLine:   4,
			EndCol:    12,
		},
		lexItem{
			Type:      itemSpaces,
			RawValue:  " ",
			Value:     " ",
			Start:     34,
			End:       35,
			StartLine: 4,
			StartCol:  12,
			EndLine:   4,
			EndCol:    13,
		},
		lexItem{
			Type:      itemBareString,
			RawValue:  "beef",
			Value:     "beef",
			Start:     35,
			End:       39,
			StartLine: 4,
			StartCol:  13,
			EndLine:   4,
			EndCol:    17,
		},
		lexItem{
			Type:      itemGreaterThanSign,
			RawValue:  ">",
			Value:     ">",
			Start:     39,
			End:       40,
			StartLine: 4,
			StartCol:  17,
			EndLine:   4,
			EndCol:    18,
		},
		lexItem{
			Type:      itemSemicolon,
			RawValue:  ";",
			Value:     ";",
			Start:     40,
			End:       41,
			StartLine: 4,
			StartCol:  18,
			EndLine:   5,
			EndCol:    0,
		},
		lexItem{
			Type:      itemSpaces,
			RawValue:  "\n\t",
			Value:     "\n\t",
			Start:     41,
			End:       43,
			StartLine: 5,
			StartCol:  0,
			EndLine:   5,
			EndCol:    2,
		},
		lexItem{
			Type:      itemBraceRight,
			RawValue:  "}",
			Value:     "}",
			Start:     43,
			End:       44,
			StartLine: 5,
			StartCol:  2,
			EndLine:   6,
			EndCol:    0,
		},
		lexItem{
			Type:      itemSpaces,
			RawValue:  "\n",
			Value:     "\n",
			Start:     44,
			End:       45,
			StartLine: 6,
			StartCol:  0,
			EndLine:   0,
			EndCol:    0,
		},
		lexItem{
			Type:  itemEOF,
			Start: 45,
			End:   45,
		},
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fail()
	}
}

func lexOneSwiftString(l *lexer) stateFn {
	r := l.next()
	switch r {
	case eof:
		return l.eof()
	default:
		l.backup()
		return lexStringSwift(lexOneSwiftString)
	}
}

func lexOneObjcString(l *lexer) stateFn {
	r := l.next()
	switch r {
	case eof:
		return l.eof()
	default:
		l.backup()
		return lexStringObjc(lexOneObjcString)
	}
}

func lexOneASCIIPlistString(l *lexer) stateFn {
	r := l.next()
	switch r {
	case eof:
		return l.eof()
	default:
		l.backup()
		return lexStringASCIIPlist(lexOneASCIIPlistString)
	}
}

func TestLexStringSwift(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{`""`, ""},
		{`"a"`, "a"},
		{`"\\\0\t\r\n\'\""`, "\\\x00\t\r\n'\""},
		{`"\u{a}"`, "\n"},
		{`"\u{0a}"`, "\n"},
		{`"\u{00a}"`, "\n"},
		{`"\u{000a}"`, "\n"},
		{`"\u{0000a}"`, "\n"},
		{`"\u{00000a}"`, "\n"},
		{`"\u{000000a}"`, "\n"},
		{`"\u{0000000a}"`, "\n"},
		{`"\u{0000000a}a"`, "\na"},
	}
	for _, c := range cases {
		l := newLexer(c.input, "", lexOneSwiftString)
		lexItems := drainLexer(&l)
		if len(lexItems) != 2 {
			t.Fail()
		} else {
			actual := lexItems[0].Value
			if c.expected != actual {
				t.Fail()
			}
		}
	}
}

func TestLexStringSwiftInvalid(t *testing.T) {
	cases := []struct {
		input string
		msg   string
	}{
		{`"`, ":1:1: unterminated string literal"},
		{`"
		`, ":1:1: unterminated string literal"},
		{`"
		"`, ":1:1: unterminated string literal"},
		{`"\b"`, ":1:1: invalid escape"},
		{`"\u"`, ":1:1: invalid unicode escape"},
		{`"\u{"`, ":1:1: invalid unicode escape"},
		{`"\u{}"`, ":1:1: invalid unicode escape"},
		{`"\u{123456789}"`, ":1:1: invalid unicode escape"},
		{`"\u{110000}"`, ":1:1: invalid unicode escape"},
	}
	for _, c := range cases {
		l := newLexer(c.input, "", lexOneSwiftString)
		lexItems := drainLexer(&l)
		if len(lexItems) != 1 {
			t.Fail()
		} else {
			err := lexItems[0].Err
			if err == nil {
				t.Fail()
			} else {
				msg := err.Error()
				if c.msg != msg {
					t.Fail()
				}
			}
		}
	}
}

func TestLexStringObjc(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{`""`, ""},

		{`"a"`, "a"},

		{`"\\\a\b\f\n\r\t\v\'\"\?"`, "\\\a\b\f\n\r\t\v'\"?"},

		{`"\u0024\u0040\u0060"`, "$@`"},
		{`"\u00a0\u00a1"`, "\u00a0\u00a1"},

		{`"\U00000024\U00000040\U00000060"`, "$@`"},
		{`"\U000000a0\U000000a1"`, "\u00a0\u00a1"},

		{`"\xa"`, "\n"},
		{`"\xag"`, "\ng"},
		{`"\x0a"`, "\n"},
		{`"\x0"`, "\u0000"},
		{`"\x1f"`, "\u001f"},

		{`"\0"`, "\u0000"},
		{`"\08"`, "\u00008"},

		{`"\00"`, "\u0000"},
		{`"\008"`, "\u00008"},

		{`"\000"`, "\u0000"},
		{`"\0008"`, "\u00008"},
	}
	for _, c := range cases {
		l := newLexer(c.input, "", lexOneObjcString)
		lexItems := drainLexer(&l)
		if len(lexItems) != 2 {
			t.Fail()
		} else {
			actual := lexItems[0].Value
			if c.expected != actual {
				t.Fail()
			}
		}
	}
}

func TestLexStringObjcInvalid(t *testing.T) {
	cases := []struct {
		input string
		msg   string
	}{
		{`"`, ":1:1: unterminated string literal"},
		{`"
		`, ":1:1: unterminated string literal"},
		{`"
		"`, ":1:1: unterminated string literal"},
		{`"\c"`, ":1:1: invalid escape"},

		{`"\u"`, ":1:1: invalid universal character name"},
		{`"\u0"`, ":1:1: invalid universal character name"},
		{`"\u00"`, ":1:1: invalid universal character name"},
		{`"\u000"`, ":1:1: invalid universal character name"},
		{`"\ug"`, ":1:1: invalid universal character name"},
		{`"\u0020"`, ":1:1: invalid universal character name"},
		{`"\ud800"`, ":1:1: invalid universal character name"},
		{`"\udfff"`, ":1:1: invalid universal character name"},

		{`"\U"`, ":1:1: invalid universal character name"},
		{`"\U0"`, ":1:1: invalid universal character name"},
		{`"\U00"`, ":1:1: invalid universal character name"},
		{`"\U000"`, ":1:1: invalid universal character name"},
		{`"\U0000"`, ":1:1: invalid universal character name"},
		{`"\U00000"`, ":1:1: invalid universal character name"},
		{`"\U000000"`, ":1:1: invalid universal character name"},
		{`"\U0000000"`, ":1:1: invalid universal character name"},
		{`"\Ug"`, ":1:1: invalid universal character name"},
		{`"\U00000020"`, ":1:1: invalid universal character name"},
		{`"\U0000d800"`, ":1:1: invalid universal character name"},
		{`"\U0000dfff"`, ":1:1: invalid universal character name"},

		{`"\x"`, ":1:1: invalid escape"},
		{`"\xg"`, ":1:1: invalid escape"},
		{`"\xa0"`, ":1:1: invalid escape"},
		{`"\xff"`, ":1:1: invalid escape"},

		{`"\200"`, ":1:1: invalid escape"},
		{`"\777"`, ":1:1: invalid escape"},
	}
	for _, c := range cases {
		l := newLexer(c.input, "", lexOneObjcString)
		lexItems := drainLexer(&l)
		if len(lexItems) != 1 {
			t.Fail()
		} else {
			err := lexItems[0].Err
			if err == nil {
				t.Fail()
			} else {
				msg := err.Error()
				if c.msg != msg {
					t.Fail()
				}
			}
		}
	}
}

func TestLexStringASCIIPlist(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{`""`, ""},

		{`"a"`, "a"},

		{`"\\\a\b\f\n\r\t\v\'\"\?"`, "\\\a\b\f\n\r\t\v'\"?"},

		{`"\000"`, "\u0000"},
		{`"\040"`, " "},
		{`"\000a"`, "\u0000a"},
		{`"\1761"`, "~1"},

		{`"\Ua"`, "\n"},
		{`"\Uag"`, "\ng"},
		{`"\UA"`, "\n"},
		{`"\U7e"`, "~"},
		{`"\U7E"`, "~"},
		{`"\U100"`, "Ā"},
		{`"\Ud7ff"`, "\ud7ff"},
		{`"\Ue800"`, "\ue800"},
		{`"\Uffff"`, "\uffff"},

		{`"\UD83E\UDD14"`, "🤔"},
	}
	for _, c := range cases {
		l := newLexer(c.input, "", lexOneASCIIPlistString)
		lexItems := drainLexer(&l)
		if len(lexItems) != 2 {
			t.Fail()
		} else {
			actual := lexItems[0].Value
			if c.expected != actual {
				t.Fail()
			}
		}
	}
}

func TestLexStringASCIIPlistInvalid(t *testing.T) {
	cases := []struct {
		input string
		msg   string
	}{
		{`"`, ":1:1: unterminated string literal"},
		{`"
		`, ":1:1: unterminated string literal"},
		{`"
		"`, ":1:1: unterminated string literal"},

		{`"\c"`, ":1:1: invalid escape"},

		{`"\Ug"`, ":1:1: invalid UTF-16 escape"},
		{`"\Ud800"`, ":1:1: invalid UTF-16 escape"},
		{`"\Ud800\Ua"`, ":1:1: invalid UTF-16 escape"},
	}
	for _, c := range cases {
		l := newLexer(c.input, "", lexOneASCIIPlistString)
		lexItems := drainLexer(&l)
		if len(lexItems) != 1 {
			t.Fail()
		} else {
			err := lexItems[0].Err
			if err == nil {
				t.Fail()
			} else {
				msg := err.Error()
				if c.msg != msg {
					t.Fail()
				}
			}
		}
	}
}
