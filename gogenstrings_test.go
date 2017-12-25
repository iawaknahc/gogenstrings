package main

import (
	"reflect"
	"testing"

	"github.com/iawaknahc/gogenstrings/xmlplist"
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

func TestXmlPlistValueToInfoPlist(t *testing.T) {
	input := xmlplist.Value{
		Value: map[string]interface{}{
			"CFBundleDevelopmentRegion": xmlplist.Value{
				Value: "$(DEVELOPMENT_LANGUAGE)",
			},
			"CFBundleDisplayName": xmlplist.Value{
				Value: "MyApp",
			},
			"NFCReaderUsageDescription": xmlplist.Value{
				Value: "Use NFC",
			},
		},
	}
	actual, err := xmlPlistValueToInfoPlist(input, "")
	expected := infoPlist{
		"CFBundleDisplayName":       "MyApp",
		"NFCReaderUsageDescription": "Use NFC",
	}
	if err != nil || !reflect.DeepEqual(actual, expected) {
		t.Fail()
	}
}

func TestInfoPlistToEntryMap(t *testing.T) {
	input := infoPlist{
		"CFBundleDisplayName":       "MyApp",
		"NFCReaderUsageDescription": "Use NFC",
	}
	actual := input.toEntryMap()
	expected := entryMap{
		"CFBundleDisplayName": entry{
			key:   "CFBundleDisplayName",
			value: "MyApp",
		},
		"NFCReaderUsageDescription": entry{
			key:   "NFCReaderUsageDescription",
			value: "Use NFC",
		},
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fail()
	}
}
