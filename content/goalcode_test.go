// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package content

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/text/textcore"
	"github.com/stretchr/testify/assert"
)

// TestBindGoal runs the compiled Goal blocks of a page in order, with the
// second one seeing the variable of the first, and checks that a block that
// does not match the compiled code goes to the interpreter, with the blocks
// that ran compiled to replay.
func TestBindGoal(t *testing.T) {
	srcs := []string{"x := 1", "fmt.Println(x + 1)", "fmt.Println(x + 2)"}
	page := &GoalPage{Keys: []string{CodeKey(srcs[0]), CodeKey(srcs[1]), CodeKey(srcs[2])}}
	page.Start = func() GoalBlock {
		return func(b *core.Frame, out io.Writer) GoalBlock {
			x := 1
			core.NewText(b).SetText("block 0")
			return func(b *core.Frame, out io.Writer) GoalBlock {
				fmt.Fprintln(out, x+1)
				return func(b *core.Frame, out io.Writer) GoalBlock {
					fmt.Fprintln(out, x+2)
					return nil
				}
			}
		}
	}
	defer func(c map[string]*GoalPage, bte func(*textcore.Editor, *core.Frame, string)) {
		CompiledGoal, BindTextEditor = c, bte
	}(CompiledGoal, BindTextEditor)
	CompiledGoal = map[string]*GoalPage{"page.md": page}
	var interpreted []string
	var replay []string
	BindTextEditor = func(ed *textcore.Editor, parent *core.Frame, language string) {
		interpreted = append(interpreted, strings.TrimSpace(ed.Lines.String()))
		if GoalReplay != nil {
			for _, r := range GoalReplay {
				replay = append(replay, strings.TrimSpace(r))
			}
			GoalReplay = nil
		}
	}

	b := core.NewBody()
	run := func(src string) *core.Frame {
		ed := textcore.NewEditor(b)
		ed.Lines.SetString(src)
		fr := core.NewFrame(b)
		BindCode(ed, fr, "Goal")
		return fr
	}
	output := func(fr *core.Frame) string {
		if !fr.HasChildren() {
			return ""
		}
		if oe, ok := fr.Child(fr.NumChildren() - 1).(*textcore.Editor); ok {
			return strings.TrimSpace(oe.Lines.String())
		}
		return ""
	}

	// all compiled
	resetGoal("page.md")
	f0, f1, f2 := run(srcs[0]), run(srcs[1]), run(srcs[2])
	assert.Equal(t, 1, f0.NumChildren())
	assert.Equal(t, "2", output(f1))
	assert.Equal(t, "3", output(f2))
	assert.Empty(t, interpreted)

	// the second block differs: it and the rest go to the interpreter,
	// which first replays the first block
	resetGoal("page.md")
	run(srcs[0])
	run("fmt.Println(x + 10)")
	run(srcs[2])
	assert.Equal(t, []string{"fmt.Println(x + 10)", srcs[2]}, interpreted)
	assert.Equal(t, []string{srcs[0]}, replay)

	// another page has no compiled Goal code
	interpreted, replay = nil, nil
	resetGoal("other.md")
	run(srcs[0])
	assert.Equal(t, []string{srcs[0]}, interpreted)
	assert.Empty(t, replay)
}
