// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on http://github.com/dmarkham/enumer and
// golang.org/x/tools/cmd/stringer:

// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package enumgen

// BuildBitFlagMethods builds methods specific to bit flag types.
func (g *Generator) BuildBitFlagMethods(runs [][]Value, typeName string) {
	g.Printf("\n")

	g.Printf(StringHasBitFlagMethod, typeName)
}

// Arguments to format are:
//
//	[1]: type name
const StringHasBitFlagMethod = `// HasBitFlag returns whether these
// bit flags have the given bit flag set.
func (i %[1]s) HasBitFlag(f enums.BitFlag) bool {
	in := int64(i)
	return atomic.LoadInt64(&in)&(1<<uint32(f.Int64())) != 0
}
`
