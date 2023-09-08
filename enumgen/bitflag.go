// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on http://github.com/dmarkham/enumer and
// golang.org/x/tools/cmd/stringer:

// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package enumgen

import "text/template"

// BuildBitFlagMethods builds methods specific to bit flag types.
func (g *Generator) BuildBitFlagMethods(runs [][]Value, typ *Type) {
	d := &TmplData{
		TypeName: typ.Name,
	}

	g.Printf("\n")

	g.ExecTmpl(StringMethodBitFlagTmpl, d)
	g.ExecTmpl(HasFlagMethodTmpl, d)
	g.ExecTmpl(SetFlagMethodTmpl, d)
}

var HasFlagMethodTmpl = template.Must(template.New("HasFlagMethod").Parse(
	`// HasFlag returns whether these
// bit flags have the given bit flag set.
func (i {{.TypeName}}) HasFlag(f enums.BitFlag) bool {
	return i&(1<<uint32(f.Int64())) != 0
}
`))

var SetFlagMethodTmpl = template.Must(template.New("SetFlagMethod").Parse(
	`// SetFlag sets the value of the given
// flags in these flags to the given value.
func (i *{{.TypeName}}) SetFlag(on bool, f ...enums.BitFlag) {
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
`))

var StringMethodBitFlagTmpl = template.Must(template.New("StringMethodBitFlag").Parse(
	`// String returns the string representation
// of this {{.TypeName}} value.
func (i {{.TypeName}}) String() string {
	str := ""
	for _, ie := range _{{.TypeName}}Values {
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
`))

var SetStringMethodBitFlagTmpl = template.Must(template.New("SetStringMethodBitFlag").Parse(
	`// SetString sets the {{.TypeName}} value from its
// string representation, and returns an
// error if the string is invalid.
func (i *{{.TypeName}}) SetString(s string) error {
	*i = 0
	flgs := strings.Split(s, "|")
	for _, flg := range flgs {
		if val, ok := _{{.TypeName}}NameToValueMap[flg]; ok {
			i.SetFlag(true, &val)
		} else if val, ok := _{{.TypeName}}NameToValueMap[strings.ToLower(flg)]; ok {
			i.SetFlag(true, &val)
		} else {
			{{.IfInvalid}}
		}
	}
	return nil
}
`))
