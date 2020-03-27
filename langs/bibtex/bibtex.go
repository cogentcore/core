// Copyright (c) 2020, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Copied and only lightly modified from:
// https://github.com/nickng/bibtex
// Licenced under an Apache-2.0 licence
// and presumably Copyright (c) 2017 by Nick Ng

package bibtex

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
)

// BibString is a segment of a bib string.
type BibString interface {
	RawString() string // Internal representation.
	String() string    // Displayed string.
}

// BibVar is a string variable.
type BibVar struct {
	Key   string    // Variable key.
	Value BibString // Variable actual value.
}

// RawString is the internal representation of the variable.
func (v *BibVar) RawString() string {
	return v.Key
}

func (v *BibVar) String() string {
	if v == nil {
		return ""
	}
	return v.Value.String()
}

// BibConst is a string constant.
type BibConst string

// NewBibConst converts a constant string to BibConst.
func NewBibConst(c string) BibConst {
	return BibConst(c)
}

// RawString is the internal representation of the constant (i.e. the string).
func (c BibConst) RawString() string {
	return fmt.Sprintf("{%s}", string(c))
}

func (c BibConst) String() string {
	return string(c)
}

// BibComposite is a composite string, may contain both variable and string.
type BibComposite []BibString

// NewBibComposite creates a new composite with one element.
func NewBibComposite(s BibString) *BibComposite {
	comp := &BibComposite{}
	return comp.Append(s)
}

// Append adds a BibString to the composite
func (c *BibComposite) Append(s BibString) *BibComposite {
	comp := append(*c, s)
	return &comp
}

func (c *BibComposite) String() string {
	var buf bytes.Buffer
	for _, s := range *c {
		buf.WriteString(s.String())
	}
	return buf.String()
}

// RawString returns a raw (bibtex) representation of the composite string.
func (c *BibComposite) RawString() string {
	var buf bytes.Buffer
	for i, comp := range *c {
		if i > 0 {
			buf.WriteString(" # ")
		}
		switch comp := comp.(type) {
		case *BibConst:
			buf.WriteString(comp.RawString())
		case *BibVar:
			buf.WriteString(comp.RawString())
		case *BibComposite:
			buf.WriteString(comp.RawString())
		}
	}
	return buf.String()
}

// BibEntry is a record of BibTeX record.
type BibEntry struct {
	Type     string
	CiteName string
	Fields   map[string]BibString
}

// NewBibEntry creates a new BibTeX entry.
func NewBibEntry(entryType string, citeName string) *BibEntry {
	spaceStripper := strings.NewReplacer(" ", "")
	cleanedType := strings.ToLower(spaceStripper.Replace(entryType))
	cleanedName := spaceStripper.Replace(citeName)
	return &BibEntry{
		Type:     cleanedType,
		CiteName: cleanedName,
		Fields:   map[string]BibString{},
	}
}

// AddField adds a field (key-value) to a BibTeX entry.
func (entry *BibEntry) AddField(name string, value BibString) {
	entry.Fields[strings.TrimSpace(name)] = value
}

// BibTex is a list of BibTeX entries.
type BibTex struct {
	Preambles []BibString          // List of Preambles
	Entries   []*BibEntry          // Items in a bibliography.
	KeyMap    map[string]*BibEntry // fast key lookup map -- made on demand in Lookup
	StringVar map[string]*BibVar   // Map from string variable to string.
}

// NewBibTex creates a new BibTex data structure.
func NewBibTex() *BibTex {
	return &BibTex{
		Preambles: []BibString{},
		Entries:   []*BibEntry{},
		StringVar: make(map[string]*BibVar),
	}
}

// AddPreamble adds a preamble to a bibtex.
func (bib *BibTex) AddPreamble(p BibString) {
	bib.Preambles = append(bib.Preambles, p)
}

// AddEntry adds an entry to the BibTeX data structure.
func (bib *BibTex) AddEntry(entry *BibEntry) {
	bib.Entries = append(bib.Entries, entry)
}

// AddStringVar adds a new string var (if does not exist).
func (bib *BibTex) AddStringVar(key string, val BibString) {
	bib.StringVar[key] = &BibVar{Key: key, Value: val}
}

// GetStringVar looks up a string by its key.
func (bib *BibTex) GetStringVar(key string) *BibVar {
	if bv, ok := bib.StringVar[key]; ok {
		return bv
	}
	// at this point, key is usually a month -- just pass through
	bib.AddStringVar(key, NewBibConst(key))
	return bib.StringVar[key]
}

// String returns a BibTex data structure as a simplified BibTex string.
func (bib *BibTex) String() string {
	var bibtex bytes.Buffer
	for _, entry := range bib.Entries {
		bibtex.WriteString(fmt.Sprintf("@%s{%s,\n", entry.Type, entry.CiteName))
		for key, val := range entry.Fields {
			if i, err := strconv.Atoi(strings.TrimSpace(val.String())); err == nil {
				bibtex.WriteString(fmt.Sprintf("  %s = %d,\n", key, i))
			} else {
				bibtex.WriteString(fmt.Sprintf("  %s = {%s},\n", key, strings.TrimSpace(val.String())))
			}
		}
		bibtex.Truncate(bibtex.Len() - 2)
		bibtex.WriteString(fmt.Sprintf("\n}\n"))
	}
	return bibtex.String()
}

// RawString returns a BibTex data structure in its internal representation.
func (bib *BibTex) RawString() string {
	var bibtex bytes.Buffer
	for k, strvar := range bib.StringVar {
		bibtex.WriteString(fmt.Sprintf("@string{%s = {%s}}\n", k, strvar.String()))
	}
	for _, preamble := range bib.Preambles {
		bibtex.WriteString(fmt.Sprintf("@preamble{%s}\n", preamble.RawString()))
	}
	for _, entry := range bib.Entries {
		bibtex.WriteString(fmt.Sprintf("@%s{%s,\n", entry.Type, entry.CiteName))
		for key, val := range entry.Fields {
			if i, err := strconv.Atoi(strings.TrimSpace(val.String())); err == nil {
				bibtex.WriteString(fmt.Sprintf("  %s = %d,\n", key, i))
			} else {
				bibtex.WriteString(fmt.Sprintf("  %s = %s,\n", key, val.RawString()))
			}
		}
		bibtex.Truncate(bibtex.Len() - 2)
		bibtex.WriteString(fmt.Sprintf("\n}\n"))
	}
	return bibtex.String()
}

// PrettyString pretty prints a BibTex.
func (bib *BibTex) PrettyString() string {
	var buf bytes.Buffer
	for i, entry := range bib.Entries {
		if i != 0 {
			fmt.Fprint(&buf, "\n")
		}
		fmt.Fprintf(&buf, "@%s{%s,\n", entry.Type, entry.CiteName)

		// Determine key order.
		keys := []string{}
		for key := range entry.Fields {
			keys = append(keys, key)
		}

		priority := map[string]int{"title": -3, "author": -2, "url": -1}
		sort.Slice(keys, func(i, j int) bool {
			pi, pj := priority[keys[i]], priority[keys[j]]
			return pi < pj || (pi == pj && keys[i] < keys[j])
		})

		// Write fields.
		tw := tabwriter.NewWriter(&buf, 1, 4, 1, ' ', 0)
		for _, key := range keys {
			value := entry.Fields[key].String()
			format := stringformat(value)
			fmt.Fprintf(tw, "    %s\t=\t"+format+",\n", key, value)
		}
		tw.Flush()

		// Close.
		buf.WriteString("}\n")
	}
	return buf.String()

}

// stringformat determines the correct formatting verb for the given BibTeX field value.
func stringformat(v string) string {
	// Numbers may be represented unquoted.
	if _, err := strconv.Atoi(v); err == nil {
		return "%s"
	}

	// Strings with certain characters must be brace quoted.
	if strings.ContainsAny(v, "\"{}") {
		return "{%s}"
	}

	// Default to quoted string.
	return "%q"
}

// MakeKeyMap creates the KeyMap from CiteName to entry
func (bib *BibTex) MakeKeyMap() {
	bib.KeyMap = make(map[string]*BibEntry, len(bib.Entries))
	for _, be := range bib.Entries {
		bib.KeyMap[be.CiteName] = be
	}
}

// Lookup finds CiteName in entries, using fast KeyMap (made on demand)
// returns nil, false if not found
func (bib *BibTex) Lookup(cite string) (*BibEntry, bool) {
	if bib.KeyMap == nil {
		bib.MakeKeyMap()
	}
	be, has := bib.KeyMap[cite]
	return be, has
}
