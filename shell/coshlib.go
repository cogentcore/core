// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shell

// CoshLib is the automatically-included cosh standard library code
var CoshLib = `
func splitLines(str string) []string {
 	return strings.Split(str, "\n")
}
`
