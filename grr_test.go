// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grr

import (
	"fmt"
	"os"
	"testing"
)

func TestNew(t *testing.T) {
	b, err := do()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(b)
}

func open(filename string) error {
	return Errorf("could not open file %q", filename)
}

func b() error {
	return New("there was a problem b")
}

func do() ([]byte, error) {
	if err := open("foo"); err != nil {
		return nil, err
	}
	if err := b(); err != nil {
		return nil, err
	}
	b, err := os.ReadFile("foo")
	if err != nil {
		return nil, Wrap(err)
	}
	return b, nil
}
