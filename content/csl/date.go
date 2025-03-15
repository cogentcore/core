// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package csl

// The CSL input model supports two different date representations:
// an EDTF string (preferred), and a more structured alternative.
type Date struct {
	DateParts [][]any `json:"date-parts,omitempty"`
	Season    any     `json:"season,omitempty"`
	Circa     string  `json:"circa,omitempty"`
	Literal   string  `json:"literal,omitempty"`
	Raw       string  `json:"raw,omitempty"`
}

func (dt *Date) Year() string {
	// todo: look in literal etc
	if len(dt.DateParts) > 0 {
		if len(dt.DateParts[0]) > 0 {
			return dt.DateParts[0][0].(string) // this is normally it
		}
	}
	return ""
}
