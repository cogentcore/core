// Copyright (c) 2023, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package fs

import "github.com/hack-pad/hackpadfs/indexeddb"

// FS represents a filesystem that implements the Node.js fs API.
type FS struct {
	indexeddb.FS
}
