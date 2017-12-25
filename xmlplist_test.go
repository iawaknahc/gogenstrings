package main

import (
	"reflect"
	"testing"
	"time"
)

func TestParseXMLPlist(t *testing.T) {
	input := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
	<dict>
		<key>key1</key>
		<array>
			<integer>-1</integer>
			<real>1.5</real>
			<string>s</string>
			<date>2017-12-25T00:00:00Z</date>
			<true/>
			<false/>
			<data>
			ab+/
			</data>
		</array>
	</dict>
</plist>
`
	actual, err := parseXMLPlist(input, "")
	expected := XMLPlistValue{
		Value: map[string]interface{}{
			"key1": XMLPlistValue{
				Value: []interface{}{
					XMLPlistValue{
						Value: int64(-1),
						Line:  7,
						Col:   4,
					},
					XMLPlistValue{
						Value: 1.5,
						Line:  8,
						Col:   4,
					},
					XMLPlistValue{
						Value: "s",
						Line:  9,
						Col:   4,
					},
					XMLPlistValue{
						Value: time.Date(2017, 12, 25, 0, 0, 0, 0, time.UTC),
						Line:  10,
						Col:   4,
					},
					XMLPlistValue{
						Value: true,
						Line:  11,
						Col:   4,
					},
					XMLPlistValue{
						Value: false,
						Line:  12,
						Col:   4,
					},
					XMLPlistValue{
						Value: []byte{105, 191, 191},
						Line:  13,
						Col:   4,
					},
				},
				Line: 6,
				Col:  3,
			},
		},
		Line: 4,
		Col:  2,
	}
	if err != nil || !reflect.DeepEqual(actual, expected) {
		t.Fail()
	}
}

func TestXMLPlistValueFlatten(t *testing.T) {
	input := XMLPlistValue{
		Value: map[string]interface{}{
			"key1": XMLPlistValue{
				Value: []interface{}{
					XMLPlistValue{
						Value: int64(-1),
						Line:  7,
						Col:   4,
					},
					XMLPlistValue{
						Value: 1.5,
						Line:  8,
						Col:   4,
					},
					XMLPlistValue{
						Value: "s",
						Line:  9,
						Col:   4,
					},
					XMLPlistValue{
						Value: time.Date(2017, 12, 25, 0, 0, 0, 0, time.UTC),
						Line:  10,
						Col:   4,
					},
					XMLPlistValue{
						Value: true,
						Line:  11,
						Col:   4,
					},
					XMLPlistValue{
						Value: false,
						Line:  12,
						Col:   4,
					},
					XMLPlistValue{
						Value: []byte{105, 191, 191},
						Line:  13,
						Col:   4,
					},
				},
				Line: 6,
				Col:  3,
			},
		},
		Line: 4,
		Col:  2,
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
