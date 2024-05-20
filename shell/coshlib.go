// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shell

// CoshLib is the automatically-included cosh standard library code
var CoshLib = `
// splitLines returns a slice of given string split by lines
func splitLines(str string) []string {
 	return strings.Split(str, "\n")
}

// fileExists returns true if given file exists
func fileExists(path string) bool {
 	ex := dirs.FileExists(path)
	return ex
}
`
