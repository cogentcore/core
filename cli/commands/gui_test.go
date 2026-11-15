// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package commands

import (
	"image"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buttonTexts returns the texts of the buttons that are the children of
// the given menu.
func buttonTexts(m *core.Scene) []string {
	var ts []string
	for _, kid := range m.Children {
		if b, ok := kid.(*core.Button); ok {
			ts = append(ts, b.Text)
		}
	}
	return ts
}

// findButton returns the button with the given text in the given tree,
// or nil if there is none.
func findButton(n tree.Node, text string) *core.Button {
	var found *core.Button
	for k := range n.AsTree().All() {
		if b, ok := k.(*core.Button); ok && b.Text == text {
			found = b
			break
		}
	}
	return found
}

// waitWindow waits for the window with the given title to be shown, and
// returns its main scene, or nil if it is not shown in time.
func waitWindow(t *testing.T, title string) *core.Scene {
	t.Helper()
	for range 2000 { // (up to 20s, for a loaded machine)
		if w := core.AllRenderWindows.FindName(title); w != nil {
			if sc := w.MainScene(); sc != nil && sc.Body != nil && sc.AsyncLockIfShown() {
				sc.AsyncUnlock()
				return sc
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Errorf("window %q is not shown", title)
	return nil
}

// closeWindow closes the window of the given scene.
func closeWindow(sc *core.Scene) {
	if sc == nil {
		return
	}
	if sc.AsyncLockIfShown() {
		defer sc.AsyncUnlock()
		sc.Close()
	}
}

// click clicks the given button with the lock of its window.
func click(b *core.Button) {
	b.AsyncLock()
	defer b.AsyncUnlock()
	b.Send(events.Click)
}

// TestMenu tests the Command menu of an app: the entries, a script, and
// the buttons to edit the commands and the choosers, and the menu of a
// nested subcommand.
func TestMenu(t *testing.T) {
	cs := testCommands(t, filepath.Join(t.TempDir(), "commands.json"))
	cs.Entries = append(cs.Entries, &Entry{Name: "psgrep", Script: psgrep()})
	require.NoError(t, cs.Save())
	m := core.NewScene()
	cs.Menu(m, func(string) {})
	ts := buttonTexts(m)
	assert.Equal(t, "Git", ts[0])
	assert.Contains(t, ts, "psgrep")
	assert.Equal(t, []string{"Edit commands", "Edit choosers"}, ts[len(ts)-2:])

	m = core.NewScene()
	cs.menu(m, nil, nil)
	assert.Empty(t, buttonTexts(m), "(no commands)")

	stash := &SubCmd{Name: "stash", Cmd: "git stash", Cmds: []*SubCmd{{Name: "stash list", Cmd: "git stash list"}, {Name: "stash show", Cmd: "git stash show"}}}
	cs.Parsed["git"].SubCmds = append(cs.Parsed["git"].SubCmds, stash)
	m = core.NewScene()
	cs.nestedMenu(m, stash, nil)
	assert.Equal(t, []string{"Stash", "List", "Show"}, buttonTexts(m))
	b := core.NewButton(core.NewScene())
	cs.configCommand(b, "git stash", nil)
	assert.NotNil(t, b.Menu, "a nested subcommand has a menu")
	cs.configCommand(b, "git commit", nil)
	assert.Nil(t, b.Menu)
	assert.Contains(t, b.Tooltip, "git commit")
	assert.Nil(t, cs.Macro("nope", MainName, "x"))
	assert.Nil(t, cs.Macro("git", "commit", "x"))
}

// TestEditCommands tests the editor of the commands, in its own window:
// adding a category and a command, which are saved and shown in the
// command bar, and resetting the commands.
func TestEditCommands(t *testing.T) {
	imagex.UpdateTestImagesFor(t) // the image is not compared
	file := filepath.Join(t.TempDir(), "commands.json")
	cs := testCommands(t, file)
	n := len(cs.Entries)
	b := core.NewBody()
	tb := cs.Bar(b, func(string) {})
	core.AssertRender(t, b, "edit-commands", func() {
		edit := findButton(tb, "")
		require.NotNil(t, edit)
		b.AsyncLock()
		cs.Edit(tb)
		b.AsyncUnlock()
		sc := waitWindow(t, "Commands")
		require.NotNil(t, sc)
		defer closeWindow(sc)
		assert.True(t, core.RecycleMainWindow(&cs.Entries), "recycled")

		click(findButton(sc, "Add category"))
		click(findButton(sc, "Add command"))
		assert.Len(t, cs.Entries, n+2)
		assert.Equal(t, "New command", cs.Entries[n+1].Name)
		saved := Open(file)
		assert.Len(t, saved.Entries, n+2, "saved")

		click(findButton(sc, "Edit script")) // nothing selected: a snackbar
		click(findButton(sc, "Reset"))       // a dialog that confirms it
		assert.Len(t, cs.Entries, n+2, "not reset until confirmed")
	})
}

// TestRunDialogWindow tests the form of a command in its own window, which
// runs its command line, and the dialog of a script that is confirmed.
func TestRunDialogWindow(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
	imagex.UpdateTestImagesFor(t) // the image is not compared
	cs := testCommands(t, filepath.Join(t.TempDir(), "commands.json"))
	ran := make(chan string, 4)
	run := func(cmdline string) { ran <- cmdline }
	b := core.NewBody()
	tb := cs.Bar(b, run)
	core.AssertRender(t, b, "run-dialog", func() {
		b.AsyncLock()
		cs.RunDialog(tb, "git commit", run)
		b.AsyncUnlock()
		d := waitWindow(t, "git commit")
		require.NotNil(t, d)
		click(findButton(d, "Run"))
		select {
		case got := <-ran:
			assert.Equal(t, "git commit", got)
		case <-time.After(20 * time.Second):
			t.Error("not run")
		}

		// a subcommand that is parsed first, a command with subcommands,
		// whose menu is shown, and a command that is not found
		prev := parseSub
		parseSub = func(cmd string) (*SubCmd, error) {
			sc := &SubCmd{Cmd: cmd, Source: cmd + " -h"}
			sc.SetFromHelp(gitCommitHelp)
			return sc, nil
		}
		t.Cleanup(func() { parseSub = prev })
		b.AsyncLock()
		cs.RunDialog(tb, "git status", run)
		b.AsyncUnlock()
		d = waitWindow(t, "git status")
		require.NotNil(t, d)
		assert.NotNil(t, cs.cachedSub("git status"), "parsed")
		closeWindow(d)
		b.AsyncLock()
		cs.RunDialog(tb, "git", run)
		cs.RunDialog(tb, "no-such-command-xyz", run)
		b.AsyncUnlock()
		time.Sleep(200 * time.Millisecond)

		s := &Script{Name: "hello", Doc: "Says hello", Confirm: true, Code: "#!/bin/sh\necho hello\n"}
		b.AsyncLock()
		cs.RunScript(tb, "", s, run)
		b.AsyncUnlock()
		select { // the confirmation dialog is shown first
		case got := <-ran:
			t.Errorf("run without confirming: %q", got)
		case <-time.After(100 * time.Millisecond):
		}

		s.Confirm = false
		b.AsyncLock()
		cs.RunScript(tb, "", s, run)
		b.AsyncUnlock()
		assert.Contains(t, <-ran, "/bin/sh", "run without confirming")
	})
}

// TestScriptWindows tests the editor of a new script of a command, and
// that of an existing one, and the editor of the choosers, in their own
// windows, and the menu of the scripts of a command.
func TestScriptWindows(t *testing.T) {
	imagex.UpdateTestImagesFor(t) // the image is not compared
	cs := testCommands(t, filepath.Join(t.TempDir(), "commands.json"))
	s := psgrep()
	require.NoError(t, cs.SaveScript("git", s, nil))
	b := core.NewBody()
	tb := cs.Bar(b, func(string) {})
	core.AssertRender(t, b, "script-windows", func() {
		m := core.NewScene()
		cs.editScriptsMenu(m, cs.cached("git"))
		sm := core.NewScene()
		m.Children[0].(*core.Button).Menu(sm, image.Point{})
		assert.Equal(t, []string{"New script", "Edit psgrep"}, buttonTexts(sm))

		b.AsyncLock()
		cs.NewScript(tb, "git", "")
		b.AsyncUnlock()
		d := waitWindow(t, "New script: git")
		require.NotNil(t, d)
		assert.Nil(t, findButton(d, "Delete script"))
		click(findButton(d, "Add argument"))
		closeWindow(d)

		b.AsyncLock()
		cs.EditScript(tb, "git", s)
		b.AsyncUnlock()
		d = waitWindow(t, "Edit script: git")
		require.NotNil(t, d)
		click(findButton(d, "Delete script"))
		assert.Empty(t, cs.cached("git").Scripts, "deleted")

		b.AsyncLock()
		cs.EditChoosers(tb)
		b.AsyncUnlock()
		d = waitWindow(t, "Choosers")
		require.NotNil(t, d)
		click(findButton(d, "Add chooser"))
		closeWindow(d)
	})
}

// TestChooserMenu tests the menu of the values of a chooser, which sets
// its text field.
func TestChooserMenu(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses printf")
	}
	imagex.UpdateTestImagesFor(t) // the image is not compared
	cs := testCommands(t, filepath.Join(t.TempDir(), "commands.json"))
	b := core.NewBody()
	tf := core.NewTextField(b).SetText("two")
	core.AssertRender(t, b, "chooser-menu", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		cs.chooserMenu(tf, tf, &Chooser{Name: "Numbers", Command: "printf 'one\\ntwo\\n'"}, image.Point{})
		cs.chooserMenu(tf, tf, &Chooser{Name: "Fails", Command: "exit 3"}, image.Point{}) // an error snackbar
		m := core.NewScene()
		chooserItems(m, tf, &Chooser{Name: "Numbers"}, []string{"one", "two"})
		assert.Equal(t, []string{"one", "two"}, buttonTexts(m))
		m.Children[0].(*core.Button).Send(events.Click)
		assert.Equal(t, "one", tf.Text())
	})
}

// TestFormWindows tests the buttons of the form of a subcommand, in its own
// window: clearing it, making a new macro and a new script of it, and
// editing and deleting its macro, and running a macro of a subcommand that
// is parsed first.
func TestFormWindows(t *testing.T) {
	imagex.UpdateTestImagesFor(t) // the image is not compared
	cs := testCommands(t, filepath.Join(t.TempDir(), "commands.json"))
	require.NoError(t, cs.SaveMacro("git", &Macro{Name: "wip", SubCmd: "commit", Values: &SubCmdArgs{Flags: map[string]string{"--message": "wip"}}}, nil))
	ran := make(chan string, 4)
	run := func(cmdline string) { ran <- cmdline }
	b := core.NewBody()
	tb := cs.Bar(b, run)
	form := func() *core.Scene {
		b.AsyncLock()
		cs.RunDialog(tb, "git commit", run)
		b.AsyncUnlock()
		d := waitWindow(t, "git commit")
		require.NotNil(t, d)
		return d
	}
	core.AssertRender(t, b, "form-windows", func() {
		d := form()
		click(findButton(d, "Clear"))
		click(findButton(d, "New macro"))
		nm := waitWindow(t, "New macro: git commit")
		require.NotNil(t, nm)
		click(findButton(nm, "Save")) // (without a name)
		closeWindow(nm)

		d = form()
		mb := findButton(d, "Macros")
		require.NotNil(t, mb)
		sm := core.NewScene()
		mb.AsyncLock()
		mb.Menu(sm, image.Point{})
		require.Equal(t, []string{"Git commit wip", "Git commit"}, buttonTexts(sm), "with the new one")
		sm.Children[0].(*core.Button).Send(events.Click) // (with the lock of the window of the menu)
		mb.AsyncUnlock()
		em := waitWindow(t, "Edit macro: git commit")
		require.NotNil(t, em)
		click(findButton(em, "Delete macro"))
		assert.Nil(t, cs.Macro("git", "commit", "wip"), "deleted")

		d = form()
		click(findButton(d, "New script"))
		ns := waitWindow(t, "New script: git")
		require.NotNil(t, ns)
		closeWindow(ns)

		prev := parseSub
		parseSub = func(cmd string) (*SubCmd, error) {
			sc := &SubCmd{Cmd: cmd, Source: cmd + " -h"}
			sc.SetFromHelp(gitCommitHelp)
			return sc, nil
		}
		t.Cleanup(func() { parseSub = prev })
		b.AsyncLock()
		cs.RunMacro(tb, "git", &Macro{Name: "st", SubCmd: "status", Values: &SubCmdArgs{Flags: map[string]string{"--verbose": "true"}}}, run)
		b.AsyncUnlock()
		select {
		case got := <-ran:
			assert.Equal(t, "git status --verbose", got)
		case <-time.After(20 * time.Second):
			t.Error("the macro is not run")
		}
		assert.Equal(t, "git status st", cs.Name("git status --verbose"), "the macro that ran it")
	})
}

// waitPopup waits for the dialog or popup of the window of the given body
// whose body has the given title (or a menu, if the title is ""), and
// returns it with the lock of the body held, which the caller must unlock.
// A dialog is the top main scene of the window, and a menu a popup.
func waitPopup(t *testing.T, b *core.Body, title string) *core.Scene {
	t.Helper()
	for range 2000 { // (up to 20s, for a loaded machine)
		b.AsyncLock()
		scs := core.Popups(b)
		if rw := b.Scene.RenderWindow(); rw != nil {
			if top := rw.MainScene(); top != nil {
				scs = append(scs, top)
			}
		}
		for _, sc := range scs {
			if (title == "" && sc.Body == nil) || (sc.Body != nil && sc.Body.Title == title) {
				return sc
			}
		}
		b.AsyncUnlock()
		time.Sleep(10 * time.Millisecond)
	}
	b.AsyncLock()
	var have []string
	for _, sc := range core.Popups(b) {
		if sc.Body != nil {
			have = append(have, sc.Body.Title)
		} else {
			have = append(have, "<"+sc.Name+">")
		}
	}
	b.AsyncUnlock()
	t.Fatalf("popup %q is not shown (popups: %q)", title, have)
	return nil
}

// TestInWindowDialogs tests the dialogs of a window: confirming a script
// and a macro, editing a script from its confirmation and from its context
// menu in the command bar, and confirming the reset of the commands in
// their editor.
func TestInWindowDialogs(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
	imagex.UpdateTestImagesFor(t) // the image is not compared
	cs := testCommands(t, filepath.Join(t.TempDir(), "commands.json"))
	s := &Script{Name: "hello", Doc: "Says hello", Confirm: true, Code: "#!/bin/sh\necho hello\n"}
	cs.Entries = append(cs.Entries, &Entry{Name: "hello", Script: s})
	require.NoError(t, cs.Save())
	ran := make(chan string, 4)
	run := func(cmdline string) { ran <- cmdline }
	b := core.NewBody()
	tb := cs.Bar(b, run)
	received := func(what string) string {
		select {
		case got := <-ran:
			return got
		case <-time.After(20 * time.Second):
			t.Errorf("%s is not run", what)
			return ""
		}
	}
	core.AssertRender(t, b, "in-window-dialogs", func() {
		// the confirmation of a script
		b.AsyncLock()
		cs.RunScript(tb, "", s, run)
		b.AsyncUnlock()
		d := waitPopup(t, b, "hello")
		core.ButtonByText(d, "Run").Send(events.Click)
		b.AsyncUnlock()
		assert.Contains(t, received("the script"), "/bin/sh")

		// editing it from its confirmation
		b.AsyncLock()
		cs.RunScript(tb, "", s, run)
		b.AsyncUnlock()
		d = waitPopup(t, b, "hello")
		core.ButtonByText(d, "Edit script").Send(events.Click)
		b.AsyncUnlock()
		closeWindow(waitWindow(t, "Edit script"))

		// and from its context menu in the bar
		bt := findButton(tb, "hello")
		require.NotNil(t, bt)
		b.AsyncLock()
		bt.ShowContextMenu(events.NewMouse(events.ContextMenu, events.Right, image.Point{}, 0))
		b.AsyncUnlock()
		m := waitPopup(t, b, "")
		edit := core.ButtonByText(m, "Edit script")
		require.NotNil(t, edit)
		edit.Send(events.Click)
		b.AsyncUnlock()
		closeWindow(waitWindow(t, "Edit script"))

		// the confirmation of a macro
		mc := &Macro{Name: "wip", SubCmd: "commit", Confirm: true, Values: &SubCmdArgs{Flags: map[string]string{"--message": "wip"}}}
		b.AsyncLock()
		cs.RunMacro(tb, "git", mc, run)
		b.AsyncUnlock()
		d = waitPopup(t, b, "Git commit wip")
		core.ButtonByText(d, "Run").Send(events.Click)
		b.AsyncUnlock()
		assert.Equal(t, "git commit --message wip", received("the macro"))

		// resetting the commands in their editor, once confirmed
		b.AsyncLock()
		cs.Edit(tb)
		b.AsyncUnlock()
		sc := waitWindow(t, "Commands")
		require.NotNil(t, sc)
		defer closeWindow(sc)
		click(findButton(sc, "Reset"))
		rd := waitPopup(t, sc.Body, "Reset commands")
		core.ButtonByText(rd, "Reset").Send(events.Click)
		sc.AsyncUnlock()
		assert.Len(t, cs.Entries, len(Standard()), "reset")
	})
}
