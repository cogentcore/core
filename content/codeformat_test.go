// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package content

import (
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/text/textcore"
	"github.com/stretchr/testify/assert"
)

func TestFormatCode(t *testing.T) {
	b := core.NewBody()
	ed := textcore.NewEditor(b)
	ed.Lines.SetString("x:=1\nif x>0 {\nfmt.Println(x)\n}\n")
	ran := ""
	ed.OnChange(func(e events.Event) { ran = ed.Lines.String() }) // as running the code
	bindFormat(ed, "Go")
	ed.SendChange()
	// the code is formatted before it runs, without changing it again
	want := "x := 1\nif x > 0 {\n\tfmt.Println(x)\n}\n"
	assert.Equal(t, want, ed.Lines.String())
	assert.Equal(t, want, ran)
	assert.False(t, ed.Lines.IsChanged())

	// code that cannot be formatted is left as it is
	ed.Lines.SetString("x := \n")
	ed.SendChange()
	assert.Equal(t, "x := \n", ed.Lines.String())
}
