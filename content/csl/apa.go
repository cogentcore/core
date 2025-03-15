// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package csl

import (
	"strings"
	"unicode"
)

// CiteAPA generates an APA-style citation string from the given item.
func CiteAPA(it *Item) string {
	switch it.Type {
	case Book, Collection, Chapter:
		return CiteAPABook(it)
	default: // articles
		return CiteAPAArticle(it)
	}
}

func CiteAPABook(it *Item) string {
	return ""
}

func CiteAPAArticle(it *Item) string {
	var w strings.Builder
	w.WriteString(NamesLastFirstInitialCommaAmpersand(it.Author))
	w.WriteString(" (" + it.Issued.Year() + "). ")
	if len(it.Title) > 0 {
		w.WriteString(it.Title)
		if !unicode.IsPunct(rune(it.Title[len(it.Title)-1])) {
			w.WriteString(".")
		}
		w.WriteString(" ")
	}
	if len(it.ContainerTitle) > 0 {
		w.WriteString(it.ContainerTitle)
		w.WriteString(", ")
	}
	if it.Volume != "" {
		w.WriteString(it.Volume)
		if it.Number != "" {
			w.WriteString("(" + it.Number + ")")
		}
		w.WriteString(", ")
	}
	if it.Page != "" {
		w.WriteString(it.Page + ".")
	}
	if it.URL != "" {
		w.WriteString(" " + it.URL)
	}
	if it.DOI != "" {
		w.WriteString(" http://doi.org/" + it.DOI)
	}
	return w.String()
}
