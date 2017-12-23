package main

type routineCall struct {
	startLine int
	startCol  int
	key       string
	comment   string
	// the first path this routine call is found
	path string
}
