// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package csl

import (
	"fmt"
	"strings"
)

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

// NameFamilyGiven returns the family and given names from given name record,
// parsing what is available if not already parsed.
// todo: add suffix stuff!
func NameFamilyGiven(nm *Name) (family, given string) {
	if nm.Family != "" && nm.Given != "" {
		return nm.Family, nm.Given
	}
	pnm := ""
	switch {
	case nm.Family != "":
		pnm = nm.Family
	case nm.Given != "":
		pnm = nm.Given
	case nm.Literal != "":
		pnm = nm.Literal
	}
	if pnm == "" {
		fmt.Printf("csl.NameFamilyGiven name format error: no valid name: %#v\n", nm)
		return
	}
	ci := strings.Index(pnm, ",")
	if ci > 0 {
		return pnm[:ci], strings.TrimSpace(pnm[ci+1:])
	}
	fs := strings.Fields(pnm)
	nfs := len(fs)
	if nfs > 1 {
		return fs[nfs-1], strings.Join(fs[:nfs-1], " ")
	}
	return pnm, ""
}

// NamesLastFirstInitialCommaAmpersand returns a list of names
// formatted as a string, in the format: Last, F., Last, F., & Last., F.
func NamesLastFirstInitialCommaAmpersand(nms []Name) string {
	var w strings.Builder
	n := len(nms)
	for i := range nms {
		nm := &nms[i]
		fam, giv := NameFamilyGiven(nm)
		w.WriteString(fam)
		if giv != "" {
			w.WriteString(", ")
			nf := strings.Fields(giv)
			for _, fn := range nf {
				w.WriteString(fn[0:1] + ".")
			}
		}
		if i == n-1 {
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

// NamesFirstInitialLastCommaAmpersand returns a list of names
// formatted as a string, in the format: A.B. Last, C.D., Last & L.M. Last
func NamesFirstInitialLastCommaAmpersand(nms []Name) string {
	var w strings.Builder
	n := len(nms)
	for i := range nms {
		nm := &nms[i]
		fam, giv := NameFamilyGiven(nm)
		if giv != "" {
			nf := strings.Fields(giv)
			for _, fn := range nf {
				w.WriteString(fn[0:1] + ".")
			}
			w.WriteString(" ")
		}
		w.WriteString(fam)
		if i == n-1 {
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

// NamesCiteEtAl returns a list of names formatted for a
// citation within a document, as Last [et al..] or
// Last & Last if exactly two authors.
func NamesCiteEtAl(nms []Name) string {
	var w strings.Builder
	n := len(nms)
	switch {
	case n == 0:
		return "(None)"
	case n == 1:
		fam, _ := NameFamilyGiven(&nms[0])
		w.WriteString(fam)
	case n == 2:
		fam, _ := NameFamilyGiven(&nms[0])
		w.WriteString(fam)
		w.WriteString(" & ")
		fam, _ = NameFamilyGiven(&nms[1])
		w.WriteString(fam)
	default:
		fam, _ := NameFamilyGiven(&nms[0])
		w.WriteString(fam)
		w.WriteString(" et al.")
	}
	return w.String()
}
