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
const StringHasBitFlagMethod = `// Has returns whether these
// bit flags have the given bit flag set.
func (i %[1]s) HasFlag(f enums.BitFlag) bool {
	return i&(1<<uint32(f.Int64())) != 0
}
`

// Arguments to format are:
//
//	[1]: type name
const StringSetBitFlagMethod = `// Set sets the value of the given
// flags in these flags to the given value.
func (i *%[1]s) SetFlag(on bool, f ...enums.BitFlag) {
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

// Arguments to format are:
//
//	[1]: type name
const StringBitFlag = `// String returns the string representation
// of this %[1]s value.
func (i %[1]s) String() string {
	str := ""
	for _, ie := range _%[1]sValues {
		if i.HasFlag(ie) {
			ies := ie.BitIndexString()
			if str == "" {
				str = ies
			} else {
				str += "|" + ies
			}
		}
	}
	return str
}
`

// Arguments to format are:
//
//	[1]: type name
const StringSetStringBitFlagMethod = `// SetString sets the %[1]s value from its
// string representation, and returns an
// error if the string is invalid.
func (i *%[1]s) SetString(s string) error {
	*i = 0
	flgs := strings.Split(s, "|")
	for _, flg := range flgs {
		if val, ok := _%[1]sNameToValueMap[flg]; ok {
			i.SetFlag(true, &val)
		} else if val, ok := _%[1]sNameToValueMap[strings.ToLower(flg)]; ok {
			i.SetFlag(true, &val)
		} else {
			return errors.New(flg+" is not a valid value for type %[1]s")
		}
	}
	return nil
}
`
