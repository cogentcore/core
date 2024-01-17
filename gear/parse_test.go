// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gear

import (
	"testing"
)

func TestParse(t *testing.T) {
	cmds := []string{"git", "go", "goki"}
	for _, cmd := range cmds {
		a := NewCmd(cmd)
		err := a.Parse()
		if err != nil {
			t.Error(err)
		}
	}
}

func TestGetHelp(t *testing.T) {
	cmds := []string{"git", "go", "goki", "ls", "mv", "cp"}
	for _, cmd := range cmds {
		a := NewCmd(cmd)
		h, err := a.GetHelp()
		if err != nil {
			t.Error(err)
		}
		if h == "" {
			t.Errorf("got empty help string for command %q", cmd)
		}
	}
}
