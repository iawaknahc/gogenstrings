package main

import (
	"testing"
)

func TestNewEntryFromRoutineCall(t *testing.T) {
	call := routineCall{}
	call.comment = ""

	e := newEntryFromRoutineCall(call)
	if e.comment != "No comment provided by engineer." {
		t.Fail()
	}

	call.comment = "Default Value"

	e = newEntryFromRoutineCall(call)
	if e.value != "Default Value" {
		t.Fail()
	}
}

func TestEntryMergeCall(t *testing.T) {
	e := entry{}
	call := routineCall{}
	call.comment = "comment"

	e = e.mergeCall(call)
	if e.comment != call.comment {
		t.Fail()
	}

	call.comment = ""
	e = e.mergeCall(call)
	if e.comment != "No comment provided by engineer." {
		t.Fail()
	}
}

func TestEntryMergeDev(t *testing.T) {
	e := entry{}
	dev := entry{
		comment: "comment",
	}
	e = e.mergeDev(dev)
	if e.comment != dev.comment {
		t.Fail()
	}

	e = entry{
		key:   "key",
		value: "key",
	}
	dev = entry{
		value: "value",
	}
	e = e.mergeDev(dev)
	if e.value == "value" {
		t.Fail()
	}
}

func TestEntryPrint(t *testing.T) {
	e := entry{
		comment: "comment",
		key:     "key",
		value:   "value",
	}
	out := `/* comment */
"key" = "value";

`
	if e.print(false) != out {
		t.Fail()
	}

	e = entry{
		comment: "",
		key:     "key",
		value:   "value",
	}
	out = `"key" = "value";

`
	if e.print(true) != out {
		t.Fail()
	}
}
