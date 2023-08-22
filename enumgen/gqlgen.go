// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on http://github.com/dmarkham/enumer and
// golang.org/x/tools/cmd/stringer:

// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package enumgen

// Arguments to format are:
//
//	[1]: type name
const gqlgenMethods = `
// MarshalGQL implements the [graphql.Marshaler] interface.
func (i %[1]s) MarshalGQL(w io.Writer) {
	w.Write([]byte(strconv.Quote(i.String())))
}

// UnmarshalGQL implements the [graphql.Unmarshaler] interface.
func (i *%[1]s) UnmarshalGQL(value any) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("%[1]s should be a string, but got a value of type %%T instead", value)
	}
	return i.SetString(str)
}
`

func (g *Generator) buildGQLGenMethods(runs [][]Value, typeName string) {
	g.Printf(gqlgenMethods, typeName)
}
