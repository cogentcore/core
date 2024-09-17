// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package goalib defines convenient utility functions for
// use in the goal shell, available with the goalib prefix.
package goalib

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/base/slicesx"
	"cogentcore.org/core/base/stringsx"
)

// SplitLines returns a slice of given string split by lines
// with any extra whitespace trimmed for each line entry.
func SplitLines(s string) []string {
	sl := stringsx.SplitLines(s)
	for i, s := range sl {
		sl[i] = strings.TrimSpace(s)
	}
	return sl
}

// FileExists returns true if given file exists
func FileExists(path string) bool {
	ex := errors.Log1(fsx.FileExists(path))
	return ex
}

// WriteFile writes string to given file with standard permissions,
// logging any errors.
func WriteFile(filename, str string) error {
	err := os.WriteFile(filename, []byte(str), 0666)
	if err != nil {
		errors.Log(err)
	}
	return err
}

// ReadFile reads the string from the given file, logging any errors.
func ReadFile(filename string) string {
	str, err := os.ReadFile(filename)
	if err != nil {
		errors.Log(err)
	}
	return string(str)
}

// ReplaceInFile replaces all occurrences of given string with replacement
// in given file, rewriting the file.  Also returns the updated string.
func ReplaceInFile(filename, old, new string) string {
	str := ReadFile(filename)
	str = strings.ReplaceAll(str, old, new)
	WriteFile(filename, str)
	return str
}

// StringsToAnys converts a slice of strings to a slice of any,
// using slicesx.ToAny.  The interpreter cannot process generics
// yet, so this wrapper is needed.  Use for passing args to
// a command, for example.
func StringsToAnys(s []string) []any {
	return slicesx.As[string, any](s)
}

// AllFiles returns a list of all files (excluding directories)
// under the given path.
func AllFiles(path string) []string {
	var files []string
	filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		files = append(files, path)
		return nil
	})
	return files
}
