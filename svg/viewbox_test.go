// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"testing"

	"cogentcore.org/core/grr"
)

func TestViewBoxParse(t *testing.T) {
	tests := []string{"none", "xMinYMin", "xMidYMin", "xMaxYMin", "xMinYMax", "xMaxYMax slice"}
	var vb ViewBox
	for _, ts := range tests {
		grr.Test(t, vb.PreserveAspectRatio.SetString(ts))
		os := vb.PreserveAspectRatio.String()
		if os != ts {
			t.Error("parse fail", os, "!=", ts)
		}
	}
}
