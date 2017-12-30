package main

import (
	"github.com/iawaknahc/gogenstrings/errors"
)

func parseDotStrings(src, filepath string) (entries, error) {
	node, err := parseASCIIPlist(src, filepath)
	if err != nil {
		return nil, err
	}

	dict, ok := node.Value.(ASCIIPlistDict)
	if !ok {
		return nil, errors.FileLineCol(
			filepath,
			node.Line,
			node.Col,
			"not in .strings format",
		)
	}

	es := entries{}
	for _, keyNode := range dict.Keys {
		key, ok := keyNode.Value.(string)
		if !ok {
			return nil, errors.FileLineCol(
				filepath,
				keyNode.Line,
				keyNode.Col,
				"unexpected token",
			)
		}

		valueNode, ok := dict.Map[keyNode]
		if !ok {
			panic("impossible")
		}

		if !ok {
			return nil, errors.FileLineCol(
				filepath,
				keyNode.Line,
				keyNode.Col,
				"unexpected token",
			)
		}

		value, ok := valueNode.Value.(string)
		if !ok {
			return nil, errors.FileLineCol(
				filepath,
				valueNode.Line,
				valueNode.Col,
				"unexpected token",
			)
		}

		e := entry{
			filepath:  filepath,
			startLine: keyNode.Line,
			startCol:  keyNode.Col,
			comment:   keyNode.CommentBefore,
			key:       key,
			value:     value,
		}
		es = append(es, e)
	}
	return es, nil
}
