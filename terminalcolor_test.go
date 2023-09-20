// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package goosi

import (
	"fmt"
	"testing"
)

func TestTerminalColor(t *testing.T) {
	c, err := TerminalColor()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("terminal color:", c)
}
