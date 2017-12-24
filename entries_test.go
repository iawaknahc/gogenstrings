package main

import (
	"reflect"
	"testing"
)

func TestEntriesSort(t *testing.T) {
	input := entries{
		entry{
			key: "a",
		},
		entry{
			key: "A",
		},
		entry{
			key: "0",
		},
	}
	expected := entries{
		entry{
			key: "0",
		},
		entry{
			key: "A",
		},
		entry{
			key: "a",
		},
	}
	actual := input.sort()
	if !reflect.DeepEqual(actual, expected) {
		t.Fail()
	}
}

func TestEntriesPrint(t *testing.T) {
	input := entries{
		entry{
			key:     "key1",
			value:   "value1",
			comment: "comment1",
		},
		entry{
			key:     "key2",
			value:   "value2",
			comment: "comment2",
		},
	}
	expected := `/* comment1 */
"key1" = "value1";

/* comment2 */
"key2" = "value2";

`
	actual := input.print(false)
	if actual != expected {
		t.Fail()
	}

	input = entries{
		entry{
			key:   "NFCReaderUsageDescription",
			value: "Use NFC",
		},
		entry{
			key:   "NSCameraUsageDescription",
			value: "Use camera",
		},
	}
	expected = `"NFCReaderUsageDescription" = "Use NFC";

"NSCameraUsageDescription" = "Use camera";

`
	actual = input.print(true)
	if expected != actual {
		t.Fail()
	}
}

func TestEntriesToEntryMap(t *testing.T) {
	input := entries{
		entry{
			key: "a",
		},
		entry{
			key: "a",
		},
	}
	actual, err := input.toEntryMap()
	if err == nil {
		t.Fail()
	}

	input = entries{
		entry{
			key: "a",
		},
		entry{
			key: "b",
		},
	}
	actual, err = input.toEntryMap()
	expected := entryMap{
		"a": entry{
			key: "a",
		},
		"b": entry{
			key: "b",
		},
	}
	if err != nil || !reflect.DeepEqual(actual, expected) {
		t.Fail()
	}
}
