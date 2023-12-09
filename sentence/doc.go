// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sentence

import "strings"

// Doc formats the given Go documentation string for an identifier with the given
// CamelCase name and intended label. It replaces the name with the label and cleans
// up trailing punctuation.
func Doc(doc, name, label string) string {
	doc = strings.ReplaceAll(doc, name, label)

	// if we only have one period, get rid of it if it is at the end
	if strings.Count(doc, ".") == 1 {
		doc = strings.TrimSuffix(doc, ".")
	}
	return doc
}
