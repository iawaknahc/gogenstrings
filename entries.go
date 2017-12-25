package main

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/iawaknahc/gogenstrings/errors"
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

func (p entries) toEntryMap() (entryMap, error) {
	em := entryMap{}
	for _, e := range p {
		if _, ok := em[e.key]; ok {
			return nil, errors.FileLineCol(
				e.filepath,
				e.startLine,
				e.startCol,
				fmt.Sprintf("duplicated key `%v`", e.key),
			)
		}
		em[e.key] = e
	}
	return em, nil
}
