package main

type entryMap map[string]entry

func (p entryMap) mergeCalls(calls map[string]routineCall) entryMap {
	output := entryMap{}
	// Copy and merge existing entry if they are still in use
	for key, entry := range p {
		if call, ok := calls[key]; ok {
			output[key] = entry.mergeCall(call)
		}
	}
	// Copy new routine call
	for key, call := range calls {
		if _, ok := output[key]; !ok {
			output[key] = newEntryFromRoutineCall(call)
		}
	}
	return output
}

func (p entryMap) mergeDev(dev entryMap) entryMap {
	output := entryMap{}
	for key, entry := range p {
		if devEntry, ok := dev[key]; ok {
			output[key] = entry.mergeDev(devEntry)
		}
	}
	for key, devEntry := range dev {
		if _, ok := output[key]; !ok {
			output[key] = devEntry
		}
	}
	return output
}

func (p entryMap) toEntries() entries {
	out := entries{}
	for _, entry := range p {
		out = append(out, entry)
	}
	return out
}
