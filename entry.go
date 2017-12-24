package main

import (
	"strings"
)

func getComment(c string) string {
	return strings.TrimSpace(c[2 : len(c)-2])
}

func getStringValue(s string) string {
	return s[1 : len(s)-1]
}

type entry struct {
	filepath  string
	startLine int
	startCol  int
	comment   string
	key       string
	value     string
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
		output += "/* " + ls.comment + " */\n"
	}
	output += `"` + ls.key + `" = "` + ls.value + `";` + "\n\n"
	return output
}
