// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ki

import (
	"bytes"
	// "fmt"
)

// key fact of Go: interface such as Ki is implicitly a pointer!

// KiPtr provides JSON marshal / unmarshal via saved PathUnique
type KiPtr struct {
	Ptr  Ki `json:"-"`
	Path string
}

// GetPath updates the Path field with the current path to the pointer
func (k *KiPtr) GetPath() {
	k.Path = k.Ptr.PathUnique()
}

// FindPtrFromPath finds and sets the Ptr value based on the current Path string -- returns true if pointer is found and non-nil
func (k *KiPtr) FindPtrFromPath(root Ki) bool {
	// fmt.Printf("finding path: %v\n", k.Path)
	if len(k.Path) == 0 {
		return false
	}
	k.Ptr = root.FindPathUnique(k.Path)
	// fmt.Printf("found: %v\n", k.Ptr)
	return k.Ptr != nil
}

// MarshalJSON gets the current path and saves only the Path directly as value of this struct
func (k KiPtr) MarshalJSON() ([]byte, error) {
	if k.Ptr == nil {
		// if true {
		b := []byte("null")
		return b, nil
	}
	k.GetPath()
	b := make([]byte, 0, len(k.Path)+8)
	// b = append(b, []byte("{\"Path\":\"")...)
	b = append(b, []byte("\"")...)
	b = append(b, []byte(k.Path)...)
	b = append(b, []byte("\"")...)
	// fmt.Printf("json out: %v\n", string(b))
	return b, nil
}

// UnarshalJSON loads the Path string directly from saved value -- KiNode must call SetKiPtrsFmPaths to actually update the pointers, based on the root object in the tree from which trees were generated, after all the initial loading has completed and the structure is all in place
func (k *KiPtr) UnmarshalJSON(b []byte) error {
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
