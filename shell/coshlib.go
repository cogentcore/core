// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shell

import (
	"os"
	"strings"

	"cogentcore.org/core/base/dirs"
	"cogentcore.org/core/base/errors"
)

// these are special functions

// SplitLines returns a slice of given string split by lines
func SplitLines(str string) []string {
	return strings.Split(str, "\n")
}

// FileExists returns true if given file exists
func FileExists(path string) bool {
	ex := errors.Log1(dirs.FileExists(path))
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
