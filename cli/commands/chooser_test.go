// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package commands

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testRepo returns a new git repository with the branches main and fix.
func testRepo(t *testing.T) string {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git")
	}
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q", "-b", "main"},
		{"-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "--allow-empty", "-m", "first"},
		{"branch", "fix"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, string(out))
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, "new.txt"), []byte("new\n"), 0o644))
	return dir
}

// TestChooserValues tests the values of the standard choosers of git
// branches and changed files, in a repository, and the error of a command
// that fails.
func TestChooserValues(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
	dir := testRepo(t)
	chs := map[string]*Chooser{}
	for _, c := range StandardChoosers() {
		chs[c.Name] = c
	}
	vals, err := chs["Git branch"].Values(dir)
	require.NoError(t, err)
	assert.Equal(t, []string{"fix", "main"}, vals)
	vals, err = chs["Git changed file"].Values(dir)
	require.NoError(t, err)
	assert.Equal(t, []string{"new.txt"}, vals)

	// the Go packages, relative to the directory
	if _, err := exec.LookPath("go"); err == nil {
		gdir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(gdir, "go.mod"), []byte("module x\n"), 0o644))
		require.NoError(t, os.MkdirAll(filepath.Join(gdir, "sub", "y"), 0o755))
		for f, pkg := range map[string]string{"x.go": "x", "sub/y/y.go": "y"} {
			require.NoError(t, os.WriteFile(filepath.Join(gdir, f), []byte("package "+pkg+"\n"), 0o644))
		}
		vals, err = chs["Go package"].Values(gdir)
		require.NoError(t, err)
		assert.Equal(t, []string{".", "./...", "./sub/y"}, vals)
	}

	vals, err = (&Chooser{Command: "printf ' a \\n\\nb\\na\\n'"}).Values(dir)
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b"}, vals, "trimmed, without empty lines and duplicates")
	_, err = (&Chooser{Command: "echo no such thing >&2; exit 1"}).Values(dir)
	assert.ErrorContains(t, err, "no such thing")
}

// TestArgChoosers tests setting the choosers of the arguments of
// subcommands, which are kept when the command is parsed again and in a
// file of another version, and the standard ones.
func TestArgChoosers(t *testing.T) {
	file := filepath.Join(t.TempDir(), "commands.json")
	cs := testCommands(t, file)
	cs.mu.Lock()
	cs.addStandardChoosersLocked()
	cs.StandardVersion = standardVersion
	cs.mu.Unlock()
	checkout := &SubCmd{Name: "checkout", Cmd: "git checkout"}
	commit := testCommit()
	assert.Equal(t, "Git branch", cs.ArgChooser(checkout, argsField), "standard")
	require.NotNil(t, cs.Chooser("Git branch"))
	assert.Nil(t, cs.Chooser(""))
	require.NoError(t, cs.SetChoosers(append(cs.choosers(), &Chooser{Name: "Message", Command: "echo wip"}), cs.ChooserArgs()))
	require.NoError(t, cs.SetArgChooser(commit, "--message", "Message"))
	require.NoError(t, cs.SetArgChooser(checkout, argsField, ""))

	cs2 := Open(file)
	assert.Equal(t, "Message", cs2.ArgChooser(commit, "--message"), "shared")
	assert.Equal(t, "", cs2.ArgChooser(checkout, argsField), "none")
	assert.Equal(t, "echo wip", cs2.Chooser("Message").Command)

	// in a file of another version
	data, err := os.ReadFile(file)
	require.NoError(t, err)
	old := filepath.Join(t.TempDir(), "commands.json")
	require.NoError(t, os.WriteFile(old, []byte(string(data[:len(data)-1])+`, "Version": 3}`), 0o644))
	cs3 := Open(old)
	assert.Equal(t, "Message", cs3.ArgChooser(commit, "--message"))

	// a file from before the choosers gets the standard ones
	before := filepath.Join(t.TempDir(), "commands.json")
	require.NoError(t, os.WriteFile(before, []byte(`{"Version": 9, "StandardVersion": 2, "Entries": [{"Command": "git"}], "Parsed": {"git": {"Cmd": "git", "Choosers": {"checkout Arguments": "Mine"}}}}`), 0o644))
	cs4 := Open(before)
	assert.Equal(t, "Mine", cs4.ArgChooser(checkout, argsField), "the user's is kept")
	assert.Equal(t, "Git branch", cs4.ArgChooser(&SubCmd{Name: "switch", Cmd: "git switch"}, argsField))
	assert.Len(t, cs4.choosers(), len(StandardChoosers()))
	assert.Equal(t, standardVersion, cs4.StandardVersion)

	// a file of version 3 gets the new version of an old standard chooser
	v3 := filepath.Join(t.TempDir(), "commands.json")
	require.NoError(t, os.WriteFile(v3, []byte(`{"Version": 9, "StandardVersion": 3, "Entries": [], "Choosers": [{"Name": "Go package", "Command": "go list ./..."}, {"Name": "Mine", "Command": "ls"}]}`), 0o644))
	cs5 := Open(v3)
	assert.Equal(t, goPackages, cs5.Chooser("Go package").Command)
	assert.Len(t, cs5.choosers(), 2, "only updated")
	cs4.mu.Lock()
	assert.Len(t, cs4.Entries, 1, "the standard entries are not added again")
	cs4.mu.Unlock()
}

// TestChooserForm tests the button of the chooser of a field of the form
// of a subcommand, which is updated when its chooser is set, and the
// menu of its values, which sets the field.
func TestChooserForm(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
	imagex.UpdateTestImagesFor(t) // the image is not compared
	dir := testRepo(t)
	cs := testCommands(t, filepath.Join(t.TempDir(), "commands.json"))
	cs.Dir = func(core.Widget) string { return dir }
	require.NoError(t, cs.SetChoosers(StandardChoosers(), nil))
	sc := testCommit()
	d := cs.NewDialog(nil, sc, func(string) {})
	d.AssertRender(t, "chooser-form", func() {
		fm := tree.ChildByType[*core.Form](d)
		require.NotNil(t, fm)
		var args, message *core.TextField
		fm.WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
			if tf, ok := cw.(*core.TextField); ok {
				switch formFieldName(tf) {
				case argsField:
					args = tf
				case "Message":
					message = tf
				}
			}
			return true
		})
		require.NotNil(t, args)
		require.NotNil(t, message)
		assert.False(t, args.TrailingIcon.IsSet())
		assert.Equal(t, "", args.Placeholder)

		require.NoError(t, cs.SetArgChooser(sc, argsField, "Git branch"))
		fm.Update()
		assert.Equal(t, icons.ArrowDropDown, args.TrailingIcon)
		require.NoError(t, cs.SetArgChooser(sc, "--message", "Git remote"))
		fm.Update()
		assert.Equal(t, "the remote of the current branch", message.Placeholder, "what empty means")
		require.NoError(t, cs.SetArgChooser(sc, "--message", ""))
		fm.Update()
		assert.False(t, message.TrailingIcon.IsSet())
		ch := cs.Chooser("Git branch")
		vals, err := ch.Values(cs.dir(d))
		require.NoError(t, err)
		m := core.NewScene()
		chooserItems(m, args, ch, vals)
		var fix *core.Button
		for _, kid := range m.Children {
			if b, ok := kid.(*core.Button); ok && b.Text == "fix" {
				fix = b
			}
		}
		require.NotNil(t, fix)
		fix.Send(events.Click)
		assert.Equal(t, "fix", reflect.ValueOf(fm.Struct).Elem().FieldByName(argsField).String(), "chosen")

		require.NoError(t, cs.SetArgChooser(sc, argsField, ""))
		fm.Update()
		assert.False(t, args.TrailingIcon.IsSet(), "none again")
	})
}

// TestMacroPromptValues tests that the values prompted for by a macro are
// those last used for it, including a switch that is turned off.
func TestMacroPromptValues(t *testing.T) {
	imagex.UpdateTestImagesFor(t) // the images are not compared
	cs := &Commands{Parsed: map[string]*Cmd{}, Args: OpenArgs(filepath.Join(t.TempDir(), "args.json"))}
	sc := testCommit()
	mc := &Macro{Name: "wip", SubCmd: "commit", Values: &SubCmdArgs{Flags: map[string]string{"--verbose": "true", "--message": "wip"}}, Prompts: []string{"--message", "--verbose"}}
	other := &Macro{Name: "other", SubCmd: "commit", Values: mc.Values, Prompts: mc.Prompts}
	ran := ""
	run := func(cmdline string) { ran = cmdline }
	prompt := func(mc *Macro, set func(v reflect.Value)) reflect.Value {
		d := cs.NewPromptDialog(nil, "git", sc, mc, run)
		var v reflect.Value
		d.AssertRender(t, "macro-prompt-values", func() {
			fm := tree.ChildByType[*core.Form](d)
			require.NotNil(t, fm)
			v = reflect.ValueOf(fm.Struct).Elem()
			if set != nil {
				set(v)
				keys := mc.promptKeys()
				require.NoError(t, cs.Args.SetMacroValues("git", mc, sc.ArgsOf(fm.Struct), keys))
				cs.runMacroLine("git", mc, sc.ArgsCommandLine(mc.withPrompted(sc, sc.ArgsOf(fm.Struct), keys)), run)
			}
		})
		return v
	}
	v := prompt(mc, nil)
	assert.Equal(t, "wip", v.FieldByName("Message").String(), "the default")
	assert.True(t, v.FieldByName("Verbose").Bool())
	prompt(mc, func(v reflect.Value) {
		v.FieldByName("Message").SetString("the fix")
		v.FieldByName("Verbose").SetBool(false)
	})
	assert.Equal(t, "git commit --message 'the fix'", ran)
	assert.Equal(t, "git commit wip", cs.Name(ran), "the macro")

	v = prompt(mc, nil)
	assert.Equal(t, "the fix", v.FieldByName("Message").String(), "the last used")
	assert.False(t, v.FieldByName("Verbose").Bool(), "off")
	v = prompt(other, nil)
	assert.Equal(t, "wip", v.FieldByName("Message").String(), "another macro has its own")
	cs2 := &Commands{Args: OpenArgs(cs.Args.file.Path)}
	assert.Equal(t, "the fix", cs2.Args.MacroValues("git", mc).Flags["--message"], "saved")
}

// TestName tests the names of command lines: their command and
// subcommand, without their arguments.
func TestName(t *testing.T) {
	cs := testCommands(t, filepath.Join(t.TempDir(), "commands.json"))
	cs.Parsed["git"].SubCmds = append(cs.Parsed["git"].SubCmds,
		&SubCmd{Name: "checkout", Cmd: "git checkout"},
		&SubCmd{Name: "stash", Cmd: "git stash", Cmds: []*SubCmd{{Cmd: "git stash list"}}})
	for cmdline, name := range map[string]string{
		"git checkout main":          "git checkout",
		"git commit -m 'the fix'":    "git commit",
		"git stash list --all":       "git stash list",
		"git stash":                  "git stash",
		"git":                        "git",
		"ls -l a b":                  "ls",
		"other thing with arguments": "other thing with",
		"go test -v ./...":           "go test",
		"core run web":               "core run web",
		"":                           "",
	} {
		assert.Equal(t, name, cs.Name(cmdline), cmdline)
	}
}

// TestChoosersDialog tests the editor of the choosers, with the table of
// the choosers of the arguments, which are set from it.
func TestChoosersDialog(t *testing.T) {
	imagex.UpdateTestImagesFor(t) // the image is not compared
	cs := testCommands(t, filepath.Join(t.TempDir(), "commands.json"))
	require.NoError(t, cs.SetChoosers(StandardChoosers(), []*ChooserArg{
		{Command: "git", SubCmd: "checkout", Arg: argsField, Chooser: "Git branch"},
		{Command: "git", SubCmd: " stash  push ", Arg: "--message", Chooser: "Git tag"},
		{Command: "ls", Arg: argsField, Chooser: "Git changed file"},
		{Command: "git", SubCmd: "add", Arg: argsField}, // (no chooser)
	}))
	assert.Equal(t, []*ChooserArg{
		{Command: "git", SubCmd: "checkout", Arg: argsField, Chooser: "Git branch"},
		{Command: "git", SubCmd: "stash push", Arg: "--message", Chooser: "Git tag"},
		{Command: "ls", Arg: argsField, Chooser: "Git changed file"},
	}, cs.ChooserArgs())
	assert.Equal(t, "Git changed file", cs.ArgChooser(&SubCmd{Name: MainName, Cmd: "ls"}, argsField))
	assert.Equal(t, "Git tag", cs.ArgChooser(&SubCmd{Name: "stash push", Cmd: "git stash push"}, "--message"))

	d := cs.NewChoosersDialog(nil)
	d.AssertRender(t, "choosers-dialog")
	require.NoError(t, cs.SetChoosers(cs.choosers(), cs.ChooserArgs()[1:]))
	assert.Equal(t, "", cs.ArgChooser(&SubCmd{Name: "checkout", Cmd: "git checkout"}, argsField), "deleted")
}
