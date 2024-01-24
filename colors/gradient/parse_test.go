// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gradient

import (
	"bytes"
	"reflect"
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/grr"
	"cogentcore.org/core/mat32"
)

func TestFromString(t *testing.T) {
	type test struct {
		str  string
		want Gradient
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

// used in multiple tests
var (
	linearTransformTest = NewLinear().
				SetTransform(mat32.Rotate2D(mat32.Pi/2)).
				SetStart(mat32.V2(0, 0)).SetEnd(mat32.V2(1, 0)).
				AddStop(colors.Gold, 0.05).
				AddStop(colors.Red, 0.95)

	radialTransformTest = NewRadial().
				SetTransform(mat32.Translate2D(0.1, 0.1).Scale(0.5, 1.75)).
				AddStop(colors.Red, 0.3).
				AddStop(colors.Blue, 0.6).
				AddStop(colors.Orange, 0.95)
)

func TestReadXML(t *testing.T) {
	type test struct {
		str  string
		want Gradient
	}
	tests := []test{
		{`<linearGradient id="myGradient">
		<stop offset="0.6" stop-color="#f31" />
		<stop offset="1.2" stop-color="#bbbff6" />
	  </linearGradient>`, NewLinear().
			SetEnd(mat32.V2(1, 0)).
			AddStop(grr.Log1(colors.FromHex("#f31")), 0.6).
			AddStop(grr.Log1(colors.FromHex("#bbbff6")), 1.2)},

		{`<linearGradient id="something" gradientTransform="rotate(90)">
		<stop offset="5%" stop-color="gold" />
		<stop offset="95%" stop-color="red" />
	  </linearGradient>`, CopyOf(linearTransformTest)},

		{`<radialGradient id="random">
			<stop offset="10%" stop-color="blue" />
			<stop offset="35%" stop-color="purple" stop-opacity="33%" />
			<stop offset="90%" stop-color="red" />
		  </radialGradient>`, NewRadial().
			AddStop(colors.Blue, 0.1).
			AddStop(colors.Purple, 0.35, 0.33).
			AddStop(colors.Red, 0.9)},

		{`<radialGradient id="h3ll0_wor1d!" gradientTransform="translate(0.1, 0.1) scale(0.5, 1.75)">
			<stop offset="30%" stop-color="red" />
			<stop offset="60%" stop-color="blue" />
			<stop offset="95%" stop-color="orange" />
		  </radialGradient>`, CopyOf(radialTransformTest)},
	}
	for _, test := range tests {
		r := bytes.NewBufferString(test.str)
		var have Gradient
		grr.Test(t, ReadXML(&have, r))
		if !reflect.DeepEqual(have, test.want) {
			t.Errorf("for %s: \n expected: \n %#v \n but got: \n %#v", test.str, test.want, have)
		}
	}
}
