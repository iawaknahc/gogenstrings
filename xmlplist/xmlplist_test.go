package xmlplist

import (
	"reflect"
	"testing"
	"time"
)

func TestXMLPlistValueFlatten(t *testing.T) {
	input := Value{
		Value: map[string]interface{}{
			"key1": Value{
				Value: []interface{}{
					Value{
						Value: int64(-1),
					},
					Value{
						Value: 1.5,
					},
					Value{
						Value: "s",
					},
					Value{
						Value: time.Date(2017, 12, 25, 0, 0, 0, 0, time.UTC),
					},
					Value{
						Value: true,
					},
					Value{
						Value: false,
					},
					Value{
						Value: []byte{105, 191, 191},
					},
				},
			},
		},
	}
	actual := input.Flatten()
	expected := map[string]interface{}{
		"key1": []interface{}{
			int64(-1),
			1.5,
			"s",
			time.Date(2017, 12, 25, 0, 0, 0, 0, time.UTC),
			true,
			false,
			[]byte{105, 191, 191},
		},
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fail()
	}
}

func TestParseXMLPlist(t *testing.T) {
	cases := []struct {
		input    string
		expected interface{}
	}{
		// string
		{"<string/>", ""},
		{"<string></string>", ""},
		{"<string>1</string>", "1"},

		// integer
		{"<integer>-1</integer>", int64(-1)},

		// real
		{"<real>1.5</real>", 1.5},

		// date
		{"<date>2017-12-25T00:00:00Z</date>", time.Date(2017, 12, 25, 0, 0, 0, 0, time.UTC)},
		// true
		{"<true></true>", true},
		{"<true/>", true},

		// false
		{"<false></false>", false},
		{"<false/>", false},

		// data
		{"<data>ab+/</data>", []byte{105, 191, 191}},
		{"<data>\t\n ab+/\t\n </data>", []byte{105, 191, 191}},

		// array
		{"<array></array>", []interface{}{}},
		{"<array><true/></array>", []interface{}{true}},

		// dict
		{"<dict></dict>", map[string]interface{}{}},
		{
			"<dict><key/><string/></dict>",
			map[string]interface{}{
				"": "",
			},
		},
	}
	for _, c := range cases {
		prefix := `<?xml version="1.0" encoding="UTF-8"?><!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd"><plist version="1.0">`
		suffix := "</plist>"
		input := prefix + c.input + suffix
		actual, err := ParseXMLPlist(input, "")
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
