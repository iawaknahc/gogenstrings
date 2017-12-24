package main

import (
	"bytes"
	"sort"
)

type entries []entry

func (p entries) sort() entries {
	out := make(entries, len(p))
	copy(out, p)
	less := func(i, j int) bool {
		return out[i].key < out[j].key
	}
	sort.SliceStable(out, less)
	return out
}

func (p entries) print(suppressEmptyComment bool) string {
	buf := bytes.Buffer{}
	for _, entry := range p {
		buf.WriteString(entry.print(suppressEmptyComment))
	}
	return buf.String()
}
