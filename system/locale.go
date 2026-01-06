// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package system

import "strings"

// Locale represents a https://www.rfc-editor.org/rfc/bcp/bcp47.txt standard
// language tag, consisting of language and region, e.g., "en-US", "fr-FR", "ja-JP".
type Locale string

// Language returns the language portion of the locale tag (e.g., en, fr, ja)
func (l Locale) Language() string {
	lang, _, _ := strings.Cut(string(l), "-")
	return lang
}

// Region returns the region portion of the locale tag (e.g., US, FR, JA)
func (l Locale) Region() string {
	_, region, _ := strings.Cut(string(l), "-")
	return region
}
