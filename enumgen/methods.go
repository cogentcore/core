// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on http://github.com/dmarkham/enumer and
// golang.org/x/tools/cmd/stringer:

// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package enumgen

import "fmt"

// Arguments to format are:
//
//	[1]: type name
const stringNameToValueMethod = `// SetString sets the enum value from its
// string representation, and returns an
// error if the string is invalid.
func (i *%[1]s) SetString(s string) error {
	if val, ok := _%[1]sNameToValueMap[s]; ok {
		*i = val
		return nil
	}

	if val, ok := _%[1]sNameToValueMap[strings.ToLower(s)]; ok {
		*i = val
		return nil
	}
	return 0, fmt.Errorf("%%s does not belong to %[1]s values", s)
}
`

// Arguments to format are:
//
//	[1]: type name
const stringValuesMethod = `// Values returns all possible values this
// enum type has. This slice will be in the
// same order as those returned by Strings and Descs.
func (i %[1]s) Values() []%[1]s {
	return _%[1]sValues
}
`

// Arguments to format are:
//
//	[1]: type name
const stringsMethod = `// Strings returns the string encodings of
// all possible values this enum type has.
// This slice will be in the same order as
// those returned by Values and Descs.
func (i %[1]s) Strings() []string {
	strs := make([]string, len(_%[1]sNames))
	copy(strs, _%[1]sNames)
	return strs
}
`

// Arguments to format are:
//
//	[1]: type name
const stringBelongsMethodLoop = `// IsValid returns whether the value is a
// valid option for its enum type.
func (i %[1]s) IsValid() bool {
	for _, v := range _%[1]sValues {
		if i == v {
			return true
		}
	}
	return false
}
`

// Arguments to format are:
//
//	[1]: type name
const stringBelongsMethodSet = `// IsValid returns whether the value is a
// valid option for its enum type.
func (i %[1]s) IsValid() bool {
	_, ok := _%[1]sMap[i] 
	return ok
}
`

// Arguments to format are:
//
//	[1]: type name
const altStringValuesMethod = `func (%[1]s) Values() []string {
	return %[1]sStrings()
}
`

func (g *Generator) buildAltStringValuesMethod(typeName string) {
	g.Printf("\n")
	g.Printf(altStringValuesMethod, typeName)
}

func (g *Generator) buildBasicExtras(runs [][]Value, typeName string, runsThreshold int) {
	// At this moment, either "g.declareIndexAndNameVars()" or "g.declareNameVars()" has been called

	// Print the slice of values
	g.Printf("\nvar _%sValues = []%s{", typeName, typeName)
	for _, values := range runs {
		for _, value := range values {
			g.Printf("\t%s, ", value.originalName)
		}
	}
	g.Printf("}\n\n")

	// Print the map between name and value
	g.printValueMap(runs, typeName, runsThreshold)

	// Print the slice of names
	g.printNamesSlice(runs, typeName, runsThreshold)

	// Print the basic extra methods
	g.Printf(stringNameToValueMethod, typeName)
	g.Printf(stringValuesMethod, typeName)
	g.Printf(stringsMethod, typeName)
	if len(runs) <= runsThreshold {
		g.Printf(stringBelongsMethodLoop, typeName)
	} else { // There is a map of values, the code is simpler then
		g.Printf(stringBelongsMethodSet, typeName)
	}
}

func (g *Generator) printValueMap(runs [][]Value, typeName string, runsThreshold int) {
	thereAreRuns := len(runs) > 1 && len(runs) <= runsThreshold
	g.Printf("\nvar _%sNameToValueMap = map[string]%s{\n", typeName, typeName)

	var n int
	var runID string
	for i, values := range runs {
		if thereAreRuns {
			runID = "_" + fmt.Sprintf("%d", i)
			n = 0
		} else {
			runID = ""
		}

		for _, value := range values {
			g.Printf("\t_%sName%s[%d:%d]: %s,\n", typeName, runID, n, n+len(value.name), value.originalName)
			g.Printf("\t_%sLowerName%s[%d:%d]: %s,\n", typeName, runID, n, n+len(value.name), value.originalName)
			n += len(value.name)
		}
	}
	g.Printf("}\n\n")
}
func (g *Generator) printNamesSlice(runs [][]Value, typeName string, runsThreshold int) {
	thereAreRuns := len(runs) > 1 && len(runs) <= runsThreshold
	g.Printf("\nvar _%sNames = []string{\n", typeName)

	var n int
	var runID string
	for i, values := range runs {
		if thereAreRuns {
			runID = "_" + fmt.Sprintf("%d", i)
			n = 0
		} else {
			runID = ""
		}

		for _, value := range values {
			g.Printf("\t_%sName%s[%d:%d],\n", typeName, runID, n, n+len(value.name))
			n += len(value.name)
		}
	}
	g.Printf("}\n\n")
}

// Arguments to format are:
//
//	[1]: type name
const jsonMethods = `
// MarshalJSON implements the json.Marshaler interface for %[1]s
func (i %[1]s) MarshalJSON() ([]byte, error) {
	return json.Marshal(i.String())
}

// UnmarshalJSON implements the json.Unmarshaler interface for %[1]s
func (i *%[1]s) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("%[1]s should be a string, got %%s", data)
	}

	var err error
	*i, err = %[1]sString(s)
	return err
}
`

func (g *Generator) buildJSONMethods(runs [][]Value, typeName string, runsThreshold int) {
	g.Printf(jsonMethods, typeName)
}

// Arguments to format are:
//
//	[1]: type name
const textMethods = `
// MarshalText implements the encoding.TextMarshaler interface for %[1]s
func (i %[1]s) MarshalText() ([]byte, error) {
	return []byte(i.String()), nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface for %[1]s
func (i *%[1]s) UnmarshalText(text []byte) error {
	var err error
	*i, err = %[1]sString(string(text))
	return err
}
`

func (g *Generator) buildTextMethods(runs [][]Value, typeName string, runsThreshold int) {
	g.Printf(textMethods, typeName)
}

// Arguments to format are:
//
//	[1]: type name
const yamlMethods = `
// MarshalYAML implements a YAML Marshaler for %[1]s
func (i %[1]s) MarshalYAML() (interface{}, error) {
	return i.String(), nil
}

// UnmarshalYAML implements a YAML Unmarshaler for %[1]s
func (i *%[1]s) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}

	var err error
	*i, err = %[1]sString(s)
	return err
}
`

func (g *Generator) buildYAMLMethods(runs [][]Value, typeName string, runsThreshold int) {
	g.Printf(yamlMethods, typeName)
}
