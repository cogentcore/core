// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package commands

import (
	"image"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/base/sharedfile"
	"cogentcore.org/core/core"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testCommit returns the parsed git commit subcommand.
func testCommit() *SubCmd {
	sc := &SubCmd{Name: "commit", Cmd: "git commit", Source: "git commit -h", Doc: "Record changes to the repository"}
	sc.SetFromHelp(gitCommitHelp)
	return sc
}

// TestMacroValues tests the form of a macro: its values and prompts, and
// the command line of a macro with prompted values.
func TestMacroValues(t *testing.T) {
	sc := testCommit()
	values := sc.FormStruct()
	v := reflect.ValueOf(values).Elem()
	v.FieldByName("Verbose").SetBool(true)
	v.FieldByName("Message").SetString("default message")
	mc := &Macro{SubCmd: "commit", Values: sc.ArgsOf(values)}

	mv := sc.MacroStruct()
	sc.SetMacroValues(mc, mv)
	m := reflect.ValueOf(mv).Elem()
	assert.Equal(t, "default message", m.FieldByName("Message").FieldByName("Value").String(), "carried over")
	m.FieldByName("Name").SetString(" wip ")
	m.FieldByName("Message").FieldByName("Prompt").SetBool(true)
	m.FieldByName("Arguments").FieldByName("Value").SetString("-- .")
	field, _ := reflect.TypeOf(mv).Elem().FieldByName("Message")
	assert.Equal(t, "-m, --message &lt;message&gt;", field.Tag.Get("label"))
	pf, _ := field.Type.FieldByName("Prompt")
	assert.Equal(t, "if checked, this will be prompted for when the macro runs", pf.Tag.Get("doc"))

	mc = sc.MacroOf(mv)
	assert.Equal(t, "wip", mc.Name)
	assert.Equal(t, []string{"--message"}, mc.Prompts)
	assert.Equal(t, "Git commit wip", mc.Label("git"))
	assert.Equal(t, "git commit --verbose --message 'default message' -- .", sc.ArgsCommandLine(mc.withPrompted(sc, nil, nil)))

	// the prompt has only the prompted arguments, with the defaults
	keys := mc.promptKeys()
	pv := sc.formStruct(keys)
	assert.Equal(t, 1, reflect.TypeOf(pv).Elem().NumField())
	mc.Values.SetValues(sc, pv)
	assert.Equal(t, "default message", reflect.ValueOf(pv).Elem().Field(0).String())
	reflect.ValueOf(pv).Elem().Field(0).SetString("the fix")
	assert.Equal(t, "git commit --verbose --message 'the fix' -- .", sc.ArgsCommandLine(mc.withPrompted(sc, sc.ArgsOf(pv), keys)))
	reflect.ValueOf(pv).Elem().Field(0).SetString("")
	assert.Equal(t, "git commit --verbose -- .", sc.ArgsCommandLine(mc.withPrompted(sc, sc.ArgsOf(pv), keys)))
}

// TestMacros tests saving and deleting macros, which are kept when their
// command is parsed again, and in a file of another version, and are at
// the top of the menu of the command.
func TestMacros(t *testing.T) {
	file := filepath.Join(t.TempDir(), "commands.json")
	cs := testCommands(t, file)
	wip := &Macro{Name: "wip", SubCmd: "commit", Values: &SubCmdArgs{Flags: map[string]string{"--verbose": "true"}}, Prompts: []string{"--message"}}
	require.NoError(t, cs.SaveMacro("git", wip, nil))
	require.NoError(t, cs.SaveMacro("ls", &Macro{Name: "long", SubCmd: MainName, Values: &SubCmdArgs{Flags: map[string]string{"-l": "true"}}}, nil))
	renamed := *wip
	renamed.Name = "work"
	require.NoError(t, cs.SaveMacro("git", &renamed, wip))
	assert.Len(t, cs.cached("git").Macros, 1)
	assert.Equal(t, "work", cs.cached("git").Macros[0].Name)

	texts := func(m *core.Scene) []string {
		var ts []string
		for _, kid := range m.Children {
			if b, ok := kid.(*core.Button); ok {
				ts = append(ts, b.Text)
			}
		}
		return ts
	}
	m := core.NewScene()
	cs.subCmdsMenu(m, "git", nil)
	assert.Equal(t, "Git commit work", texts(m)[0])
	m = core.NewScene()
	cs.subCmdsMenu(m, "ls", nil)
	assert.Equal(t, []string{"Ls long", "Ls", "Scripts"}, texts(m), "ls has a menu with its macros")

	// another process, with a file of another version
	b, err := os.ReadFile(file)
	require.NoError(t, err)
	old := filepath.Join(t.TempDir(), "commands.json")
	require.Contains(t, string(b), `"Version": 9`)
	require.NoError(t, os.WriteFile(old, []byte(strings.Replace(string(b), `"Version": 9`, `"Version": 2`, 1)), 0o644))
	cs2 := &Commands{file: sharedfile.File[commandsData]{Path: old}, Parsed: map[string]*Cmd{}}
	cs2.reload()
	require.NotNil(t, cs2.cached("git"))
	assert.False(t, cs2.cached("git").IsParsed(), "parsed again")
	assert.Equal(t, "work", cs2.cached("git").Macros[0].Name, "kept")

	require.NoError(t, cs.DeleteMacro("git", &renamed))
	assert.Empty(t, cs.cached("git").Macros)
}

// TestMacroDialogs renders the dialog of a new macro and the prompt of a
// macro.
func TestMacroDialogs(t *testing.T) {
	imagex.UpdateTestImagesFor(t)
	cs := &Commands{Parsed: map[string]*Cmd{}, Args: OpenArgs("")}
	sc := testCommit()
	mc := &Macro{Name: "wip", SubCmd: "commit", Values: &SubCmdArgs{Args: "-- .", Flags: map[string]string{"--verbose": "true", "--message": "wip"}}, Prompts: []string{"--message"}}
	d := cs.NewMacroDialog(nil, "git", sc, mc, nil, func(string) {})
	d.AssertRender(t, "macro")
	ran := ""
	d = cs.NewPromptDialog(nil, "git", sc, mc, func(cmdline string) { ran = cmdline })
	d.AssertRender(t, "macro-prompt", func() {
		fm := tree.ChildByType[*core.Form](d)
		require.NotNil(t, fm)
		reflect.ValueOf(fm.Struct).Elem().Field(0).SetString("the fix")
		fm.Update()
	})
	assert.Equal(t, "", ran)
}

// TestContext tests the context of the dialogs of the commands: the scene
// of the window of an item of a menu, which closes when it is clicked, and
// a command bar for an item of a closed menu (which crashed making a macro
// from the dialog of a subcommand in a menu).
func TestContext(t *testing.T) {
	imagex.UpdateTestImagesFor(t) // the image is not compared
	cs := &Commands{Parsed: map[string]*Cmd{}, Entries: Standard()}
	b := core.NewBody()
	tb := cs.Bar(b, nil)
	b.AssertRender(t, "context", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		st := core.NewMenu(func(m *core.Scene, pos image.Point) { core.NewButton(m).SetText("Item") }, tb, image.Point{})
		st.Run()
		item := st.Scene.Child(0).(core.Widget)
		assert.Equal(t, core.Widget(b.Scene), cs.context(item), "the scene of the window")
		assert.Equal(t, core.Widget(tb), cs.context(tb))
		st.ClosePopup()
		assert.Equal(t, core.Widget(tb), cs.context(item), "a command bar")
	})
}

// TestDialogFocus tests that the first field of the forms of the dialogs
// is in focus when they are shown: the first prompt of a macro, the name
// of a macro, and the arguments of a subcommand.
func TestDialogFocus(t *testing.T) {
	imagex.UpdateTestImagesFor(t) // the images are not compared
	cs := &Commands{Parsed: map[string]*Cmd{}, Args: OpenArgs("")}
	sc := testCommit()
	mc := &Macro{Name: "wip", SubCmd: "commit", Values: &SubCmdArgs{Flags: map[string]string{"--message": "wip"}}, Prompts: []string{"--file", "--message"}}
	for _, c := range []struct {
		name, field string
		d           *core.Body
	}{
		{"prompt", "File", cs.NewPromptDialog(nil, "git", sc, mc, func(string) {})},
		{"macro", "Name", cs.NewMacroDialog(nil, "git", sc, mc, nil, func(string) {})},
		{"form", "Arguments", cs.NewDialog(nil, sc, func(string) {})},
	} {
		c.d.AssertRender(t, "focus-"+c.name, func() {
			focused := ""
			for range 50 {
				c.d.AsyncLock()
				c.d.WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
					if _, ok := cw.(*core.TextField); ok && cwb.StateIs(states.Focused) {
						focused = cwb.Name
					}
					return tree.Continue
				})
				c.d.AsyncUnlock()
				if focused != "" {
					break
				}
				time.Sleep(20 * time.Millisecond)
			}
			assert.Equal(t, "value-"+c.field+"-string", focused, c.name)
		})
	}
}

// TestFieldWidths tests that the text fields of the forms of the dialogs,
// and of the inline forms of the arguments of macros, are wide, with no
// maximum width.
func TestFieldWidths(t *testing.T) {
	imagex.UpdateTestImagesFor(t)
	cs := &Commands{Parsed: map[string]*Cmd{}, Args: OpenArgs("")}
	sc := testCommit()
	mc := &Macro{Name: "wip", SubCmd: "commit", Prompts: []string{"--message"}}
	for name, d := range map[string]*core.Body{
		"form":  cs.NewDialog(nil, sc, func(string) {}),
		"macro": cs.NewMacroDialog(nil, "git", sc, mc, nil, func(string) {}),
	} {
		d.AssertRender(t, "field-widths-"+name, func() {
			d.AsyncLock() // (not while it is styled)
			defer d.AsyncUnlock()
			n := 0
			d.WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
				if tf, ok := cw.(*core.TextField); ok {
					n++
					assert.Equal(t, FieldMinWidth, tf.Styles.Min.X.Value, name)
					assert.Equal(t, units.UnitCh, tf.Styles.Min.X.Unit, name)
					assert.Zero(t, tf.Styles.Max.X.Value, name)
				}
				return tree.Continue
			})
			assert.Greater(t, n, 3, name)
		})
	}
}

// TestStandardMacros tests that the flags of the standard macros are those
// of their subcommands, as parsed from their help (for the commands that
// are installed), and their command lines.
func TestStandardMacros(t *testing.T) {
	if testing.Short() {
		t.Skip("parses the help of many commands")
	}
	cs := &Commands{file: sharedfile.File[commandsData]{Path: filepath.Join(t.TempDir(), "commands.json")}, Parsed: map[string]*Cmd{}}
	for root, mcs := range StandardMacros() {
		if _, err := exec.LookPath(root); err != nil {
			continue
		}
		for _, mc := range mcs {
			sc, err := cs.SubCmd(mc.Cmd(root), false)
			if !assert.NoError(t, err, mc.Label(root)) {
				continue
			}
			if mc.Values != nil {
				for fl := range mc.Values.Flags {
					assert.True(t, slices.ContainsFunc(sc.Flags, func(f *Flag) bool { return f.Name == fl }), "%s: %s", mc.Label(root), fl)
				}
			}
			for _, p := range mc.Prompts {
				assert.True(t, p == argsField || slices.ContainsFunc(sc.Flags, func(f *Flag) bool { return f.Name == p }), "%s: prompt %s", mc.Label(root), p)
			}
		}
	}
	gc, err := cs.SubCmd("git commit", false)
	if err == nil {
		mc := StandardMacros()["git"][4]
		assert.Equal(t, "Git commit", mc.Label("git"))
		keys := mc.promptKeys()
		pv := gc.formStruct(keys)
		reflect.ValueOf(pv).Elem().Field(0).SetString("the fix")
		assert.Equal(t, "git commit --message 'the fix' --all", gc.ArgsCommandLine(mc.withPrompted(gc, gc.ArgsOf(pv), keys)))
	}
}

// TestMacroConfirm tests that a macro to be confirmed, without prompts,
// does not run until it is confirmed, unlike one that is not, and that
// Confirm is in the form of a macro.
func TestMacroConfirm(t *testing.T) {
	imagex.UpdateTestImagesFor(t) // the image is not compared
	cs := testCommands(t, filepath.Join(t.TempDir(), "commands.json"))
	ran := []string{}
	run := func(cmdline string) { ran = append(ran, cmdline) }
	b := core.NewBody()
	core.NewText(b).SetText("Main")
	b.AssertRender(t, "macro-confirm", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		cs.RunMacro(b, "git", &Macro{SubCmd: "commit", Name: "now", Values: &SubCmdArgs{Flags: map[string]string{"--verbose": "true"}}}, run)
		assert.Equal(t, []string{"git commit --verbose"}, ran)
		cs.RunMacro(b, "git", &Macro{SubCmd: "commit", Name: "sure", Confirm: true}, run)
		assert.Len(t, ran, 1, "not until confirmed")
	})

	sc := testCommit()
	mv := sc.MacroStruct()
	sc.SetMacroValues(&Macro{SubCmd: "commit", Confirm: true}, mv)
	assert.True(t, sc.MacroOf(mv).Confirm)
}

// TestOpenStandardMacros tests that the standard macros are added to new
// commands, and to those of a file from before them, keeping the macros of
// the user, including those of the same subcommand and name.
func TestOpenStandardMacros(t *testing.T) {
	cs := Open(filepath.Join(t.TempDir(), "commands.json"))
	cs.mu.Lock()
	n := len(cs.Parsed["git"].Macros)
	cs.mu.Unlock()
	assert.Equal(t, len(StandardMacros()["git"]), n)

	file := filepath.Join(t.TempDir(), "commands.json")
	old := `{"Version": 9, "StandardVersion": 1, "Entries": [{"Command": "git"}], "Parsed": {"git": {"Cmd": "git", "SubCmds": [{"Name": "status", "Cmd": "git status"}], "Macros": [{"SubCmd": "status", "Values": {"Flags": {"--short": "true"}}}, {"SubCmd": "commit", "Name": "mine"}]}}}`
	require.NoError(t, os.WriteFile(file, []byte(old), 0o644))
	cs = Open(file)
	cs.mu.Lock()
	defer cs.mu.Unlock()
	mcs := cs.Parsed["git"].Macros
	assert.Equal(t, len(StandardMacros()["git"])+1, len(mcs), "and mine")
	assert.Equal(t, "true", mcs[0].Values.Flags["--short"], "the user's version of git status is kept")
	assert.Equal(t, standardVersion, cs.StandardVersion)
}
