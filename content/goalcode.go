// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package content

import (
	"fmt"
	"io"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/text/textcore"
)

// GoalBlock is a Goal code block compiled into the app at build time
// (see cogentcore.org/core/content/cmd/contentcode). It runs the code
// with the given parent widget, which the code knows as b, writes what
// the code prints to out, and returns the next Goal code block of its
// page (nil after the last one). The next block runs in the scope of
// this one, so that it sees the variables and functions this one
// declares, as in the Goal interpreter, which runs all the Goal code of a
// page in one interpreter.
type GoalBlock func(b *core.Frame, out io.Writer) GoalBlock

// GoalPage is the Goal code of a page compiled into the app.
type GoalPage struct {

	// Keys are the [CodeKey]s of the sources of the Goal code blocks of
	// the page, in order.
	Keys []string

	// Start returns the first code block of a new run of the page,
	// in a new scope.
	Start func() GoalBlock
}

// CompiledGoal maps the name of a page file (as in [bcontent.Page.Filename])
// to its Goal code compiled into the app. When a page is shown, its Goal
// code blocks run without an interpreter as long as they match the
// compiled blocks, in order.
var CompiledGoal = map[string]*GoalPage{}

// GoalReplay are the sources of the Goal code blocks of the current page
// that ran compiled before the first one that runs in the interpreter
// (because it was edited or not compiled). An interpreter that makes a new
// Goal interpreter for the page (see [BindTextEditor]) first runs these
// sources, without showing anything, and clears them, so that the
// interpreted code sees the variables of the blocks before it, as it
// would if the whole page had run in the interpreter.
var GoalReplay []string

// AddCompiledGoal adds the given compiled pages to [CompiledGoal].
// Generated code calls it.
func AddCompiledGoal(pages map[string]*GoalPage) {
	for k, p := range pages {
		CompiledGoal[k] = p
	}
}

// goalRun is the state of the Goal code of the page being shown.
var goalRun struct {

	// page is the compiled Goal code of the page, if any.
	page *GoalPage

	// next is the next compiled block to run, if any.
	next GoalBlock

	// ran are the sources of the blocks that ran compiled, in order.
	ran []string

	// interpreted is set once a block of the page runs in the
	// interpreter: all later ones run there too.
	interpreted bool
}

// resetGoal starts the Goal code of a new page, from the given file.
func resetGoal(filename string) {
	goalRun.page = CompiledGoal[filename]
	goalRun.next = nil
	if goalRun.page != nil {
		goalRun.next = goalRun.page.Start()
	}
	goalRun.ran = nil
	goalRun.interpreted = false
	GoalReplay = nil
}

// bindGoal runs the Goal code block of the given editor: compiled, if it
// is the next compiled block of the page, and otherwise (and once it is
// edited) in the interpreter.
func bindGoal(ed *textcore.Editor, parent *core.Frame) {
	src := ed.Lines.String()
	gr := &goalRun
	n := len(gr.ran)
	if gr.interpreted || gr.next == nil || n >= len(gr.page.Keys) || gr.page.Keys[n] != CodeKey(src) {
		interpretGoal(ed, parent)
		return
	}
	f := gr.next
	gr.next = nil
	gr.ran = append(gr.ran, src)
	interpreted := false
	ed.OnChange(func(e events.Event) {
		if interpreted {
			return
		}
		if BindTextEditor == nil {
			core.ErrorSnackbar(ed, fmt.Errorf("this app has no interpreter to run edited code (only the code compiled into it)"), "Error running Goal code")
			return
		}
		// BindTextEditor runs the code now and on later changes.
		interpreted = true
		interpretGoal(ed, parent)
	})
	runCode(ed, parent, "Goal", func(out io.Writer) {
		gr.next = f(parent, out)
	})
}

// interpretGoal runs the Goal code block of the given editor in the
// interpreter of the page, after the blocks that ran compiled.
func interpretGoal(ed *textcore.Editor, parent *core.Frame) {
	if BindTextEditor == nil {
		errors.Log(fmt.Errorf("content: Goal code block was not compiled into the app and there is no interpreter to run it"))
		return
	}
	gr := &goalRun
	if !gr.interpreted {
		gr.interpreted = true
		gr.next = nil
		GoalReplay = gr.ran
	}
	BindTextEditor(ed, parent, "Goal")
}
