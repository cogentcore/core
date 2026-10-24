// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js && !goscript

package gpu

import "github.com/oliverbestmann/webgpu/wgpu"

// readBytesInto copies bytes read from a GPU buffer into dest, as
// readToSlice does.
func readBytesInto[E any](dest []E, src []byte) {
	copy(wgpu.ToBytes(dest), src)
}
