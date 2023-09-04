// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package nptime

import (
	"testing"
	"time"
)

func TestTime(t *testing.T) {
	tn := time.Now()
	tp := Time{}
	tp.SetTime(tn)
	tnr := tp.Time()

	if !tn.Equal(tnr) {
		t.Errorf("time was not reconstructed properly: %v vs. %v\n", tn, tnr)
	}
}
