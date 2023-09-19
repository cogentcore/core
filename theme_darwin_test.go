// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build darwin

package goosi

import (
	"fmt"
	"testing"
)

func TestIsDark(t *testing.T) {
	isDark, err := IsDark()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("IsDark:", isDark)
}

func TestMonitor(t *testing.T) {
	isDark, err := IsDark()
	if err != nil {
		t.Fatal(err)
	}
	ec, err := Monitor(func(b bool) {
		fmt.Println("IsDark changed to:", isDark)
	})
	if err != nil {
		t.Fatal(err)
	}
	err = <-ec
	if err != nil {
		t.Fatal(err)
	}
}
