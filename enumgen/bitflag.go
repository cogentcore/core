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
	g.Printf(StringSetBitFlagMethod, typeName)
}

// Arguments to format are:
//
//	[1]: type name
const StringHasBitFlagMethod = `// HasBitFlag returns whether these
// bit flags have the given bit flag set.
func (i *%[1]s) HasBitFlag(f enums.BitFlag) bool {
	return atomic.LoadInt64((*int64)(i))&(1<<uint32(f.Int64())) != 0
}
`

// Arguments to format are:
//
//	[1]: type name
const StringSetBitFlagMethod = `// HasBitFlag returns whether these
// bit flags have the given bit flag set.
func (i *%[1]s) SetBitFlag(on bool, f ...enums.BitFlag) {
	var mask int64
	for _, v := range f {
		mask |= 1 << v.Int64()
	}
	in := int64(*i)
	if on {
		in |= mask
		atomic.StoreInt64((*int64)(i), in)
	} else {
		in &^= mask
		atomic.StoreInt64((*int64)(i), in)
	}
}
`
