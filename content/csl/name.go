// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package csl

import "strings"

// Name represents a persons name.
type Name struct {
	Family              string `json:"family,omitempty"`
	Given               string `json:"given,omitempty"`
	DroppingParticle    string `json:"dropping-particle,omitempty"`
	NonDroppingParticle string `json:"non-dropping-particle,omitempty"`
	Suffix              string `json:"suffix,omitempty"`
	CommaSuffix         any    `json:"comma-suffix,omitempty"`
	StaticOrdering      any    `json:"static-ordering,omitempty"`
	Literal             string `json:"literal,omitempty"`
	ParseNames          any    `json:"parse-names,omitempty"`
}

// NamesList returns a list of names formatted as a string,
// in the format: Last, F., Last, F., & Last., F.
func NamesLastFirstInitialCommaAmpersand(nms []Name) string {
	var w strings.Builder
	n := len(nms)
	for i := range nms {
		nm := &nms[i]
		w.WriteString(nm.Family)
		if nm.Given != "" {
			w.WriteString(", ")
			nf := strings.Fields(nm.Given)
			for _, fn := range nf {
				w.WriteString(fn[0:1] + ".")
			}
		}
		if n == 1 || i == n-1 {
			break
		}
		if i == n-2 {
			w.WriteString(", & ")
		} else {
			w.WriteString(", ")
		}
	}
	return w.String()
}
