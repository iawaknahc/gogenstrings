package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func isSourceCodeFile(fullpath string) bool {
	ext := filepath.Ext(fullpath)
	return ext == ".swift" || ext == ".m" || ext == ".h"
}

func findLprojs(root string) (output []string, outerr error) {
	walkFn := func(fullpath string, info os.FileInfo, err error) error {
		if err != nil {
			outerr = err
			return err
		}
		if info.IsDir() {
			if strings.HasSuffix(fullpath, ".lproj") && !strings.HasSuffix(fullpath, "Base.lproj") {
				output = append(output, fullpath)
			}
		}
		return nil
	}
	filepath.Walk(root, walkFn)
	return
}

func findSourceFiles(root string, exclude *regexp.Regexp) (output []string, outerr error) {
	walkFn := func(fullpath string, info os.FileInfo, err error) error {
		if err != nil {
			outerr = err
			return err
		}
		if info.Mode().IsRegular() && isSourceCodeFile(fullpath) {
			if exclude == nil || !exclude.MatchString(fullpath) {
				output = append(output, fullpath)
			}
		}
		return nil
	}
	filepath.Walk(root, walkFn)
	return
}
