package main

import (
	"reflect"
	"testing"
)

func TestASCIIPlistNodeFlatten(t *testing.T) {
	cases := []struct {
		input    ASCIIPlistNode
		expected interface{}
	}{
		{
			ASCIIPlistNode{
				Value: "s",
			},
			"s",
		},
		{
			ASCIIPlistNode{
				Value: []byte{1},
			},
			[]byte{1},
		},
		{
			ASCIIPlistNode{
				Value: []ASCIIPlistNode{
					ASCIIPlistNode{
						Value: "s",
					},
					ASCIIPlistNode{
						Value: []byte{1},
					},
					ASCIIPlistNode{
						Value: map[ASCIIPlistNode]ASCIIPlistNode{
							ASCIIPlistNode{Value: "key"}: ASCIIPlistNode{
								Value: []ASCIIPlistNode{},
							},
						},
					},
				},
			},
			[]interface{}{
				"s",
				[]byte{1},
				map[string]interface{}{
					"key": []interface{}{},
				},
			},
		},
	}
	for _, c := range cases {
		actual := c.input.Flatten()
		if !reflect.DeepEqual(actual, c.expected) {
			t.Fail()
		}
	}
}

func TestParseASCIIPlistInvalid(t *testing.T) {
	cases := []struct {
		input string
		msg   string
	}{
		{"a=b;a=c;", ":1:5: duplicated key `a`"},
	}
	for _, c := range cases {
		_, err := parseASCIIPlist(c.input, "")
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

func TestParseASCIIPlist(t *testing.T) {
	cases := []struct {
		input    string
		expected interface{}
	}{
		// string
		{"/*a*/ a /*a*/", "a"},
		{"/*a*/ $-_.:/ /*a*/", "$-_.:/"},
		{`/*a*/"a"/*a*/`, "a"},

		// data
		{"/*a*/<>/*a*/", []byte{}},
		{"/*a*/<00>/*a*/", []byte{0}},
		{"/*a*/<0001>/*a*/", []byte{0, 1}},

		// array
		{"/*a*/(/*a*/)/*a*/", []interface{}{}},
		{"/*a*/(/*a*/1 /*a*/)/*a*/", []interface{}{"1"}},
		{"/*a*/(/*a*/1 /*a*/,/*a*/2 /*a*/)/*a*/", []interface{}{"1", "2"}},

		// dict
		{"", map[string]interface{}{}},
		{" ", map[string]interface{}{}},
		{"/*a*/", map[string]interface{}{}},
		{"/*a*/ ", map[string]interface{}{}},
		{" /*a*/", map[string]interface{}{}},
		{" /*a*/ ", map[string]interface{}{}},
		{
			"/*a*/$-_.:/ /*a*/=/*a*/a /*a*/;/*a*/",
			map[string]interface{}{
				"$-_.:/": "a",
			},
		},
		{
			`/*a*/{/*a*/"$-_.:/"/*a*/=/*a*/"$-_.:/"/*a*/;/*a*/}/*a*/`,
			map[string]interface{}{
				"$-_.:/": "$-_.:/",
			},
		},

		// nested
		{
			`/* a */{
				/* a */
				version /* a */ = 1 /* a */;
				classes /* a */ = () /* a */;
				data /* a */ = <> /* a */;
				objects /* a */ = {
					john /* a */ = doe /* a */;
					alice /* a */ = (
						{
							name = alice;
						} /* a */,
						<deadbeef> /* a */
					) /* a */;
				} /* a */;
			}/* a */`,
			map[string]interface{}{
				"version": "1",
				"classes": []interface{}{},
				"data":    []byte{},
				"objects": map[string]interface{}{
					"john": "doe",
					"alice": []interface{}{
						map[string]interface{}{
							"name": "alice",
						},
						[]byte{222, 173, 190, 239},
					},
				},
			},
		},
	}
	for _, c := range cases {
		actual, err := parseASCIIPlist(c.input, "")
		if err != nil {
			t.Fail()
		} else {
			actualFlattened := actual.Flatten()
			if !reflect.DeepEqual(actualFlattened, c.expected) {
				t.Fail()
			}
		}
	}
}
