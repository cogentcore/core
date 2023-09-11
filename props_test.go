// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ki

import (
	"encoding/json"
	"testing"
)

var PropsTest = Props{
	"intprop":    -17,
	"floatprop":  3.1415,
	"stringprop": "type string",
	"#subprops": Props{
		"sp1": "#FFE",
		"sp2": 42.2,
	},
}

func TestPropsJSonSave(t *testing.T) {
	b, err := json.MarshalIndent(PropsTest, "", "  ")
	if err != nil {
		t.Error(err)
		// } else {
		// 	fmt.Printf("props json output:\n%v\n", string(b))
	}

	tstload := make(Props)
	err = json.Unmarshal(b, &tstload)
	if err != nil {
		t.Error(err)
		// } else {
		// 	// tstb, _ := json.MarshalIndent(tstload, "", "  ")
		// fmt.Printf("props test loaded json output:\n%v\n", string(tstb))
		// because of the map randomization, this is not testable, do it manually..
		// if !bytes.Equal(tstb, b) {
		// 	t.Error("props original and unmarshal'd json rep are not equivalent")
		// }
	}
}
