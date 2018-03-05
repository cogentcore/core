// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Ki is the base element of GoKi Trees
// Ki = Tree in Japanese, and "Key" in English
package ki

import (
	"bytes"
	"fmt"
)

// key fact of Go: interface such as Ki is implicitly a pointer!

// KiPtr provides JSON marshal / unmarshal via saved PathUnique
type KiPtr struct {
	Ptr  Ki `json:"-"`
	Path string
}

func (k *KiPtr) GetPath() {
	k.Path = k.Ptr.PathUnique()
}

func (k *KiPtr) FindPtrFromPath(top Ki) bool {
	if len(k.Path) == 0 {
		return false
	}
	fmt.Printf("finding path: %v\n", k.Path)
	k.Ptr = top.FindPathUnique(k.Path)
	return k.Ptr != nil
}

// this saves type information for each object in a slice, and the unmarshal uses it to create
// proper object types
func (k KiPtr) MarshalJSON() ([]byte, error) {
	if k.Ptr == nil {
		// if true {
		b := []byte("null")
		return b, nil
	}
	k.GetPath()
	b := make([]byte, 0, len(k.Path)+4)
	b = append(b, []byte("\"")...)
	b = append(b, []byte(k.Path)...)
	b = append(b, []byte("\"")...)
	// fmt.Printf("json out: %v\n", string(b))
	return b, nil
}

func (k *KiPtr) UnmarshalJSON(b []byte) error {
	if bytes.Equal(b, []byte("null")) {
		k.Ptr = nil
		k.Path = ""
		return nil
	}
	k.Path = string(b)
	return nil
}
