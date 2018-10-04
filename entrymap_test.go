package main

import (
	"reflect"
	"testing"
)

func TestEntryMapMergeCalls(t *testing.T) {
	em := entryMap{
		"key1": entry{
			comment: "comment1",
			key:     "key1",
			value:   "value1",
		},
		"key2": entry{
			comment: "comment2_old",
			key:     "key2",
			value:   "value2",
		},
	}
	calls := map[string]routineCall{
		"key2": routineCall{
			comment: "comment2_new",
			key:     "key2",
		},
		"key3": routineCall{
			comment: "comment3",
			key:     "key3",
		},
	}
	expected := entryMap{
		"key2": entry{
			comment: "comment2_new",
			key:     "key2",
			value:   "value2",
		},
		"key3": entry{
			comment: "comment3",
			key:     "key3",
			value:   "comment3",
		},
	}
	actual := em.mergeCalls(calls)
	if !reflect.DeepEqual(actual, expected) {
		t.Fail()
	}
}

func TestEntryMapMergeDev(t *testing.T) {
	em := entryMap{
		"unused_key": entry{},
		"key1": entry{
			key:   "key1",
			value: "key1",
		},
		"key2": entry{
			key:   "key2",
			value: "value2_ja",
		},
	}
	dev := entryMap{
		"key1": entry{
			key:   "key1",
			value: "value1_en",
		},
		"key2": entry{
			key:   "key2",
			value: "value2_en",
		},
	}
	expected := entryMap{
		"key1": entry{
			key:   "key1",
			value: "key1",
		},
		"key2": entry{
			key:   "key2",
			value: "value2_ja",
		},
	}
	actual := em.mergeDev(dev)
	if !reflect.DeepEqual(actual, expected) {
		t.Fail()
	}
}

func TestEntryMapToEntries(t *testing.T) {
	input := entryMap{
		"a": entry{
			key: "a",
		},
		"A": entry{
			key: "A",
		},
		"0": entry{
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
	actual := input.toEntries()
	if len(actual) != len(expected) {
		t.Fail()
	}
}
