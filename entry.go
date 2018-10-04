package main

import (
	"strings"
)

type entry struct {
	filepath  string
	startLine int
	startCol  int
	comment   string
	key       string
	value     string
}

func newEntryFromRoutineCall(rc routineCall) entry {
	comment := "No comment provided by engineer."
	value := ""
	if rc.comment != "" {
		value = rc.comment
		comment = rc.comment
	} else {
		value = rc.key
	}
	ls := entry{
		comment: comment,
		key:     rc.key,
		value:   value,
	}
	return ls
}

func (ls entry) mergeCall(rc routineCall) entry {
	ls.comment = rc.comment
	if ls.comment == "" {
		ls.comment = "No comment provided by engineer."
	}
	return ls
}

func (ls entry) mergeDev(dev entry) entry {
	ls.comment = dev.comment
	if ls.value == ls.key {
		ls.value = dev.value
	}
	return ls
}

func (ls entry) print(suppressEmptyComment bool) string {
	output := ""
	printComment := !suppressEmptyComment || ls.comment != ""
	if printComment {
		output += "/* " + strings.TrimSpace(ls.comment) + " */\n"
	}
	output += PrintASCIIPlistString(ls.key)
	output += " = "
	output += PrintASCIIPlistString(ls.value)
	output += ";\n\n"
	return output
}
