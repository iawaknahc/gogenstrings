package main

import (
	"reflect"
	"testing"
)

func TestParseRoutineCalls(t *testing.T) {
	routineName := "NSLocalizedString"
	input := `
import Foundation
#if SOME_COMPILER_FLAG
#endif
class MyView: UILabel {
	func bind() {
		self.text = NSLocalizedString("key1", comment: "comment")
		self.text = NSLocalizedString("key2", comment: "comment")
		self.text = NSLocalizedString(@"key1", @"comment")
		self.text = NSLocalizedString("key1", @"comment")
		self.text = NSLocalizedString(@"key1", "comment")
		self.text = NSLocalizedString("key1", "comment")
	}
}
`
	expected := routineCallSlice{
		routineCall{
			filepath:  ".swift",
			startLine: 7,
			startCol:  15,
			key:       "key1",
			comment:   "comment",
		},
		routineCall{
			filepath:  ".swift",
			startLine: 8,
			startCol:  15,
			key:       "key2",
			comment:   "comment",
		},
		routineCall{
			filepath:  ".swift",
			startLine: 9,
			startCol:  15,
			key:       "key1",
			comment:   "comment",
		},
		routineCall{
			filepath:  ".swift",
			startLine: 10,
			startCol:  15,
			key:       "key1",
			comment:   "comment",
		},
		routineCall{
			filepath:  ".swift",
			startLine: 11,
			startCol:  15,
			key:       "key1",
			comment:   "comment",
		},
		routineCall{
			filepath:  ".swift",
			startLine: 12,
			startCol:  15,
			key:       "key1",
			comment:   "comment",
		},
	}
	actual, err := parseRoutineCalls(input, routineName, ".swift")
	if err != nil {
		t.Fail()
	} else if !reflect.DeepEqual(actual, expected) {
		t.Fail()
	}
}

func TestRoutineCallSliceToMap(t *testing.T) {
	input := routineCallSlice{
		routineCall{},
	}
	actual, err := input.toMap()
	if err == nil {
		t.Fail()
	}

	input = routineCallSlice{
		routineCall{
			key:     "a",
			comment: "1",
		},
		routineCall{
			key:     "a",
			comment: "2",
		},
	}
	actual, err = input.toMap()
	if err == nil {
		t.Fail()
	}

	input = routineCallSlice{
		routineCall{
			key:     "a",
			comment: "1",
		},
	}
	actual, err = input.toMap()
	expected := map[string]routineCall{
		"a": routineCall{
			key:     "a",
			comment: "1",
		},
	}
	if err != nil || !reflect.DeepEqual(actual, expected) {
		t.Fail()
	}
}
