// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gradient

import (
	"bytes"
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
		{"radial-gradient(circle at center, red 0, blue, green 100%)", NewRadial().
			AddStop(colors.Red, 0).
			AddStop(colors.Blue, 0.5).
			AddStop(colors.Green, 1)},
		{"radial-gradient(ellipse at right, purple 0.3, yellow 60%, gray)", NewRadial().
			SetCenter(mat32.V2(1, 0.5)).SetFocal(mat32.V2(1, 0.5)).
			AddStop(colors.Purple, 0.3).
			AddStop(colors.Yellow, 0.6).
			AddStop(colors.Gray, 1)},
	}
	for _, test := range tests {
		have, err := FromString(test.str)
		grr.Test(t, err)
		if !reflect.DeepEqual(have, test.want) {
			t.Errorf("for %q: \n expected: \n %#v \n but got: \n %#v", test.str, test.want, have)
		}
	}
}

func TestReadXML(t *testing.T) {
	type test struct {
		str  string
		want image.Image
	}
	tests := []test{
		{`<linearGradient id="myGradient">
		<stop offset="5%" stop-color="gold" />
		<stop offset="95%" stop-color="red" />
	  </linearGradient>`, NewLinear().
			SetStart(mat32.V2(0, 0)).SetEnd(mat32.V2(1, 0)).
			AddStop(colors.Gold, 0.05).
			AddStop(colors.Red, 0.95)},
	}
	for _, test := range tests {
		r := bytes.NewBufferString(test.str)
		have, err := ReadXML(r)
		grr.Test(t, err)
		if !reflect.DeepEqual(have, test.want) {
			t.Errorf("for %s: \n expected: \n %#v \n but got: \n %#v", test.str, test.want, have)
		}
	}
}
