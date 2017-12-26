package main

import (
	"reflect"
	"testing"
)

func TestASCIIPlistValueFlatten(t *testing.T) {
	cases := []struct {
		input    ASCIIPlistValue
		expected interface{}
	}{
		{
			ASCIIPlistValue{
				Value: "s",
			},
			"s",
		},
		{
			ASCIIPlistValue{
				Value: []byte{1},
			},
			[]byte{1},
		},
		{
			ASCIIPlistValue{
				Value: []interface{}{
					ASCIIPlistValue{
						Value: "s",
					},
					ASCIIPlistValue{
						Value: []byte{1},
					},
					ASCIIPlistValue{
						Value: map[string]interface{}{
							"key": ASCIIPlistValue{
								Value: []interface{}{},
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

func TestParseASCIIPlist(t *testing.T) {
	cases := []struct {
		input    string
		expected interface{}
	}{
		// string
		{"a", "a"},
		{"$-_.:/", "$-_.:/"},
		{`"a"`, "a"},

		// data
		{"<>", []byte{}},
		{"<00>", []byte{0}},
		{"<0001>", []byte{0, 1}},

		// array
		{"()", []interface{}{}},
		{"(1)", []interface{}{"1"}},
		{"(1,2)", []interface{}{"1", "2"}},

		// dict
		{"", map[string]interface{}{}},
		{
			"$-_.:/=a;",
			map[string]interface{}{
				"$-_.:/": "a",
			},
		},
		{
			`{"$-_.:/"="$-_.:/";}`,
			map[string]interface{}{
				"$-_.:/": "$-_.:/",
			},
		},

		// nested
		{
			`{
				version = 1;
				classes = ();
				data = <>;
				objects = {
					john = doe;
					alice = (
						{
							name = alice;
						},
						<deadbeef>
					);
				};
			}`,
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
