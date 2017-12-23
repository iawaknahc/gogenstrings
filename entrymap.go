package main

import (
	"sort"
)

type entryMap map[string]entry

func (lss entryMap) mergeCalls(rcs map[string]routineCall) entryMap {
	output := entryMap{}
	// Copy and merge existing entry if they are still in use
	for key, ls := range lss {
		if rc, ok := rcs[key]; ok {
			output[key] = ls.mergeCall(rc)
		}
	}
	// Copy new routine call
	for key, rc := range rcs {
		if _, ok := output[key]; !ok {
			output[key] = entry{
				comment: rc.comment,
				key:     rc.key,
				value:   rc.key,
			}
		}
	}
	return output
}

func (lss entryMap) mergeDev(dev entryMap) entryMap {
	output := entryMap{}
	for key, ls := range lss {
		if devLs, ok := dev[key]; ok {
			output[key] = ls.mergeDev(devLs)
		}
	}
	for key, devLs := range dev {
		if _, ok := output[key]; !ok {
			output[key] = devLs
		}
	}
	return output
}

func (lss entryMap) sort() []entry {
	slice := []entry{}
	for _, ls := range lss {
		slice = append(slice, ls)
	}
	less := func(i, j int) bool {
		return slice[i].key < slice[j].key
	}
	sort.SliceStable(slice, less)
	return slice
}
