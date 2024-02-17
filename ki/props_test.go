// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ki

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

var PropsTest = map[string]any{
	"intprop":    -17,
	"floatprop":  3.1415,
	"stringprop": "type string",
	"#subprops": map[string]any{
		"sp1": "#FFE",
		"sp2": 42.2,
	},
}

func TestPropsJSonSave(t *testing.T) {
	props := NewProps()
	props.MSet(PropsTest)
	b, err := props.MarshalJSON()
	assert.NoError(t, err)
	newProps := NewProps()
	assert.NoError(t, newProps.UnmarshalJSON(b))
	//todo add more test
}
