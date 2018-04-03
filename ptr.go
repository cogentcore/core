// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ki

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/rcoreilly/goki/ki/kit"
)

// key fact of Go: interface such as Ki is implicitly a pointer!

// Ptr provides JSON marshal / unmarshal via saved PathUnique
type Ptr struct {
	Ptr  Ki `json:"-" xml:"-"`
	Path string
}

var KiT_Ptr = kit.Types.AddType(&Ptr{}, nil)

// reset the pointer to nil, and the path to empty
func (k *Ptr) Reset() {
	k.Ptr = nil
	k.Path = ""
}

// GetPath updates the Path field with the current path to the pointer
func (k *Ptr) GetPath() {
	if k.Ptr != nil {
		k.Path = k.Ptr.PathUnique()
	} else {
		k.Path = ""
	}
}

// FindPtrFromPath finds and sets the Ptr value based on the current Path string -- returns true if pointer is found and non-nil
func (k *Ptr) FindPtrFmPath(root Ki) bool {
	// fmt.Printf("finding path: %v\n", k.Path)
	if len(k.Path) == 0 {
		k.Ptr = nil
		return true
	}
	k.Ptr = root.FindPathUnique(k.Path)
	// fmt.Printf("found: %v\n", k.Ptr)
	return k.Ptr != nil
}

// UpdatePath replaces any occurrence of oldPath with newPath, optionally only at the start of the path (typically true)
func (k *Ptr) UpdatePath(oldPath, newPath string, startOnly bool) {
	if startOnly {
		if strings.HasPrefix(k.Path, oldPath) {
			k.Path = newPath + strings.TrimPrefix(k.Path, oldPath)
		}
	} else {
		k.Path = strings.Replace(k.Path, oldPath, newPath, 1) // only do 1 replacement
	}
}

// MarshalJSON gets the current path and saves only the Path directly as value of this struct
func (k Ptr) MarshalJSON() ([]byte, error) {
	if k.Ptr == nil {
		b := []byte("null")
		return b, nil
	}
	k.GetPath()
	b := make([]byte, 0, len(k.Path)+8)
	b = append(b, []byte("\"")...)
	b = append(b, []byte(k.Path)...)
	b = append(b, []byte("\"")...)
	return b, nil
}

// UnarshalJSON loads the Path string directly from saved value -- KiNode must call SetPtrsFmPaths to actually update the pointers, based on the root object in the tree from which trees were generated, after all the initial loading has completed and the structure is all in place
func (k *Ptr) UnmarshalJSON(b []byte) error {
	// fmt.Printf("attempt to load path: %v\n", string(b))
	if bytes.Equal(b, []byte("null")) {
		k.Ptr = nil
		k.Path = ""
		return nil
	}
	k.Path = string(b)
	// fmt.Printf("loaded path: %v\n", k.Path)
	return nil
}

// MarshalXML getes the current path and saves it
func (k Ptr) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	tokens := []xml.Token{start}
	if k.Ptr == nil {
		tokens = append(tokens, xml.CharData("null"))
	} else {
		k.GetPath()
		tokens = append(tokens, xml.CharData(k.Path))
	}
	tokens = append(tokens, xml.EndElement{start.Name})
	for _, t := range tokens {
		err := e.EncodeToken(t)
		if err != nil {
			return err
		}
	}
	err := e.Flush()
	if err != nil {
		return err
	}
	return nil
}

// UnmarshalXML loads the Path string directly from saved value -- KiNode must call SetPtrsFmPaths to actually update the pointers, based on the root object in the tree from which trees were generated, after all the initial loading has completed and the structure is all in place
func (k *Ptr) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	t, err := d.Token()
	if err != nil {
		return err
	}
	ct, ok := t.(xml.CharData)
	if ok {
		tn := string(bytes.TrimSpace([]byte(ct)))
		if tn == "null" {
			// fmt.Printf("loading path: %v\n", tn)
			k.Ptr = nil
			k.Path = ""
		} else {
			// fmt.Printf("loading path: %v\n", tn)
			k.Path = tn
		}
		t, err := d.Token()
		if err != nil {
			return err
		}
		et, ok := t.(xml.EndElement)
		if ok {
			if et.Name != start.Name {
				return fmt.Errorf("ki.Ptr UnmarshalXML: EndElement: %v does not match StartElement: %v", et.Name, start.Name)
			}
			return nil
		}
		return fmt.Errorf("ki.Ptr UnmarshalXML: Token: %+v is not expected EndElement", et)
	}
	return fmt.Errorf("ki.Ptr UnmarshalXML: Token: %+v is not expected EndElement", ct)
}
