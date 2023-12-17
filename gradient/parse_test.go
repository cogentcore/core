// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gradient

import (
	"image"
	"reflect"
	"testing"

	"goki.dev/colors"
	"goki.dev/grr"
	"goki.dev/mat32/v2"
)

func TestFromString(t *testing.T) {
	type test struct {
		str  string
		want image.Image
	}
	tests := []test{
		{"linear-gradient(#e66465, #9198e5)", NewLinear().
			AddStop(grr.Log1(colors.FromHex("#e66465")), 0).
			AddStop(grr.Log1(colors.FromHex("#9198e5")), 1)},
		{"linear-gradient(to left, blue, purple, red)", NewLinear().
			SetStart(mat32.V2(1, 0)).SetEnd(mat32.V2(0, 0)).
			AddStop(colors.Blue, 0).
			AddStop(colors.Purple, 0.5).
			AddStop(colors.Red, 1)},
		{"linear-gradient(0deg, blue, green 40%, red)", NewLinear().
			SetStart(mat32.V2(0, 1)).SetEnd(mat32.V2(0, 0)).
			AddStop(colors.Blue, 0).
			AddStop(colors.Green, 0.4).
			AddStop(colors.Red, 1)},
	}
	for _, test := range tests {
		have, err := FromString(test.str)
		grr.Test(t, err)
		if !reflect.DeepEqual(have, test.want) {
			t.Errorf("for %q: \n expected: \n %#v \n but got: \n %#v", test.str, test.want, have)
		}
	}
}
