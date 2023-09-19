// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package goosi

import (
	"fmt"
	"testing"
	"time"
)

func TestThemeIsDark(t *testing.T) {
	isDark, err := IsDark()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("IsDark:", isDark)
}

func TestMonitorTheme(t *testing.T) {
	// t.Skip("comment this out to monitor theme changes (which uses a function that will never return)")
	done := make(chan struct{})
	ec, err := IsDarkMonitor(func(isDark bool) {
		fmt.Println("IsDark changed to:", isDark)
	}, done)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		time.Sleep(time.Second * 10)
		close(done)
		done <- struct{}{}

	}()
	err = <-ec
	if err != nil {
		t.Fatal(err)
	}
}
