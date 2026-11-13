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
	"strings"
	"testing"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/core"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// psgrep is a script with one argument, as in the commands of the user.
func psgrep() *Script {
	return &Script{
		Name: "psgrep",
		Doc:  "Lists the running processes whose lines match a pattern",
		Code: "#!/bin/sh\nps uaxwww | grep -i -- \"$1\" | grep -v grep\n",
		Args: []ScriptArg{{Name: "Process", Doc: "the name, or part of it, of the processes to list"}},
	}
}

// gitprune is a Goal script without arguments, confirmed, which deletes
// the branches merged into main, as in the commands of the user.
func gitprune() *Script {
	return &Script{
		Name:    "gitprune",
		Doc:     "Deletes the branches merged into main",
		Confirm: true,
		Code: `#!/usr/bin/env goal
for _, b := range goalib.SplitLines($git branch --merged main$) {
	b = strings.TrimSpace(b)
	if b != "* main" && b != "main" {
		git branch -D {b}
	}
}
`,
	}
}

// runScriptLine runs the given command line of a script with the shell,
// in the given directory, returning its output.
func runScriptLine(t *testing.T, cmdline, dir string) string {
	t.Helper()
	cmd := exec.Command("/bin/sh", "-c", cmdline)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	return string(out)
}

// TestScriptCommandLine tests the command lines of scripts: their
// arguments are passed in their order, with empty ones for switches that
// are off, and any characters, through the launcher.
func TestScriptCommandLine(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
	cs := testCommands(t, filepath.Join(t.TempDir(), "my commands", "commands.json"))
	s := &Script{
		Name: "show args",
		Code: "#!/bin/bash\nfor a in \"$@\"; do echo \"[$a]\"; done",
		Args: []ScriptArg{{Name: "Message"}, {Name: "Force", Kind: ArgSwitch, On: "--force"}, {Name: "All", Kind: ArgSwitch}, {Name: "File", Kind: ArgFile}},
	}
	values := map[string]string{"Message": `it's "a" $HOME test`, "Force": "", "All": "true", "File": "a b.txt"}
	cmdline, err := cs.ScriptCommandLine("git", s, values)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(cmdline, "/bin/sh '"), cmdline)
	out := runScriptLine(t, cmdline, t.TempDir())
	assert.Equal(t, "[it's \"a\" $HOME test]\n[]\n[true]\n[a b.txt]\n", out)
	assert.Equal(t, `'show args' 'it'\''s "a" $HOME test' '' true 'a b.txt'`, s.CommandLine(values))

	// no #! line: the shell
	s = &Script{Name: "plain", Code: "echo plain $1"}
	cmdline, err = cs.ScriptCommandLine("", s, nil)
	require.NoError(t, err)
	assert.Equal(t, "plain\n", runScriptLine(t, cmdline, t.TempDir()))
}

// TestScriptPsgrep tests running psgrep for the process of the test.
func TestScriptPsgrep(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
	cs := testCommands(t, filepath.Join(t.TempDir(), "commands.json"))
	self := filepath.Base(os.Args[0])
	cmdline, err := cs.ScriptCommandLine("", psgrep(), map[string]string{"Process": self})
	require.NoError(t, err)
	assert.Contains(t, runScriptLine(t, cmdline, t.TempDir()), self)
}

// TestScriptGoal tests running a Goal script, gitprune, in a repository.
func TestScriptGoal(t *testing.T) {
	if _, err := exec.LookPath("goal"); err != nil {
		t.Skip("goal is not installed")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	repo := t.TempDir()
	git := func(args ...string) string {
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, string(out))
		return string(out)
	}
	git("init", "-q", "-b", "main")
	git("commit", "-q", "--allow-empty", "-m", "init")
	git("branch", "merged")
	git("checkout", "-q", "-b", "unmerged")
	git("commit", "-q", "--allow-empty", "-m", "more")
	git("checkout", "-q", "main")

	cs := testCommands(t, filepath.Join(t.TempDir(), "commands.json"))
	cmdline, err := cs.ScriptCommandLine("git", gitprune(), nil)
	require.NoError(t, err)
	assert.Contains(t, runScriptLine(t, cmdline, repo), "Deleted branch merged")
	assert.Equal(t, []string{"main", "unmerged"}, strings.Fields(strings.ReplaceAll(git("branch"), "*", "")))
}

// TestScriptForm tests the form of the arguments of a script.
func TestScriptForm(t *testing.T) {
	s := &Script{Args: []ScriptArg{{Name: "Process"}, {Name: "process", Kind: ArgSwitch}, {Name: "In file", Kind: ArgFile, Doc: "the <file>"}}}
	form := s.FormStruct()
	typ := reflect.TypeOf(form).Elem()
	require.Equal(t, 3, typ.NumField())
	assert.Equal(t, reflect.TypeFor[string](), typ.Field(0).Type)
	assert.Equal(t, reflect.TypeFor[bool](), typ.Field(1).Type)
	assert.Equal(t, reflect.TypeFor[core.Filename](), typ.Field(2).Type)
	assert.Equal(t, "In file", typ.Field(2).Tag.Get("label"))
	assert.Equal(t, "the &lt;file&gt;", typ.Field(2).Tag.Get("doc"))
	values := map[string]string{"Process": "go", "process": "true", "In file": "/x"}
	s.SetFormValues(form, values)
	assert.Equal(t, values, s.FormValues(form))

	assert.Equal(t, fileinfo.Goal, gitprune().Language())
	assert.Equal(t, fileinfo.Bash, psgrep().Language())
	assert.Equal(t, fileinfo.Python, (&Script{Code: "#!/usr/bin/env python3\n"}).Language())
	assert.Equal(t, fileinfo.Bash, (&Script{Code: "echo"}).Language())
}

// TestScripts tests saving and deleting the scripts of a command and of
// entries, which are in the menus, kept when the command is parsed again,
// and shared through the file, with the values of their arguments.
func TestScripts(t *testing.T) {
	file := filepath.Join(t.TempDir(), "commands.json")
	cs := testCommands(t, file)
	cs.Args = OpenArgs(strings.TrimSuffix(file, ".json") + "-args.json")
	require.NoError(t, cs.SaveScript("git", gitprune(), nil))
	require.NoError(t, cs.SaveScript("", psgrep(), nil))
	assert.Error(t, cs.SaveScript("", &Script{}, nil), "no name")

	// another process
	cs2 := Open(file)
	gp := cs2.Script("git", "gitprune")
	require.NotNil(t, gp)
	assert.Equal(t, gitprune(), gp)
	ps := cs2.Script("", "psgrep")
	require.NotNil(t, ps)
	assert.Equal(t, "psgrep", cs2.Entries[len(cs2.Entries)-1].Label())
	assert.False(t, cs2.Entries[len(cs2.Entries)-1].IsCategory())

	// the values of the arguments
	assert.Equal(t, map[string]string{"Process": ""}, cs2.Args.ScriptValues("", ps))
	require.NoError(t, cs.Args.SetScriptValues("", ps, map[string]string{"Process": "goal"}))
	assert.Equal(t, map[string]string{"Process": "goal"}, cs2.Args.ScriptValues("", ps))

	// editing, in place
	ed := ps.clone()
	ed.Name = "pgrep"
	require.NoError(t, cs2.SaveScript("", ed, ps))
	assert.Nil(t, cs2.Script("", "psgrep"))
	assert.Equal(t, "pgrep", cs2.Entries[len(cs2.Entries)-1].Label())
	require.NoError(t, cs2.DeleteScript("", ed))
	assert.Nil(t, cs.Script("", "pgrep"), "in the other process")

	// in the menu of git, after the macros, and kept when parsed again
	menuTexts := func(m *core.Scene) []string {
		var texts []string
		for _, kid := range m.Children {
			if bt, ok := kid.(*core.Button); ok {
				texts = append(texts, bt.Text)
			}
		}
		return texts
	}
	m := core.NewScene()
	cs.subCmdsMenu(m, "git", nil)
	texts := menuTexts(m)
	assert.Equal(t, []string{"Git remote", "gitprune", "Status", "Commit", "Scripts"}, texts[len(texts)-5:], "after the macros")
	cs.mu.Lock()
	cm := cs.Parsed["git"]
	cs.mu.Unlock()
	require.NoError(t, cs.update(func() { cs.Parsed["git"] = &Cmd{Cmd: "git", SubCmds: cm.SubCmds, Scripts: cm.Scripts} }))
	assert.NotNil(t, cs.Script("git", "gitprune"))
	require.NoError(t, cs.DeleteScript("git", gp))
	assert.Nil(t, cs2.Script("git", "gitprune"))
}

// TestScriptDialogs renders the editor of a script, and the prompt of its
// arguments, which runs it.
func TestScriptDialogs(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
	imagex.UpdateTestImagesFor(t)
	cs := testCommands(t, filepath.Join(t.TempDir(), "commands.json"))
	s := psgrep()
	s.Args = append(s.Args, ScriptArg{Name: "All", Kind: ArgSwitch, Doc: "whether to list all", On: "--all"})
	d := cs.NewScriptDialog(nil, "", s, s)
	d.AssertRender(t, "script")
	ran := ""
	run := func(cmdline string) { ran = cmdline }
	d = cs.NewScriptPromptDialog(nil, "", s, run)
	var form any
	d.AssertRender(t, "script-prompt", func() {
		fm := tree.ChildByType[*core.Form](d)
		require.NotNil(t, fm)
		form = fm.Struct
		reflect.ValueOf(form).Elem().Field(0).SetString("goal")
		reflect.ValueOf(form).Elem().Field(1).SetBool(true)
		fm.Update()
	})
	assert.Equal(t, "psgrep goal --all", s.CommandLine(s.FormValues(form)))
	cs.runScript(d, "", s, s.FormValues(form), run) // (as Run does)
	assert.True(t, strings.HasPrefix(ran, "/bin/sh "), "%q", ran)
	assert.Equal(t, map[string]string{"Process": "goal", "All": "true"}, cs.Args.ScriptValues("", s), "saved")
	launcher, err := os.ReadFile(strings.Trim(strings.TrimPrefix(ran, "/bin/sh "), "'"))
	require.NoError(t, err)
	assert.Contains(t, string(launcher), "goal --all\n")
}
