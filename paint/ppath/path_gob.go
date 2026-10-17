// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js

package ppath

import (
	"bytes"
	"encoding/gob"
)

// The gob methods are not in web builds, where encoding/gob would
// otherwise be a noticeable part of the size of every app.

// GobEncode implements the gob interface.
func (p Path) GobEncode() ([]byte, error) {
	b := bytes.Buffer{}
	enc := gob.NewEncoder(&b)
	if err := enc.Encode(p); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// GobDecode implements the gob interface.
func (p *Path) GobDecode(b []byte) error {
	dec := gob.NewDecoder(bytes.NewReader(b))
	return dec.Decode(p)
}
