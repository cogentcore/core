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
	"runtime"
	"testing"
	"time"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/base/sharedfile"
	"cogentcore.org/core/core"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// gitCommitHelp is part of the output of git commit -h.
const gitCommitHelp = `usage: git commit [-a | --interactive | --patch] [-s] [-v] [-u[<mode>]] [--amend]
                  [--dry-run] [(-c | -C | --squash) <commit> | --fixup [(amend|reword):]<commit>]

    -q, --[no-]quiet      suppress summary after successful commit
    -v, --[no-]verbose    show diff in commit message template

Commit message options
    -F, --[no-]file <file>
                          read message from file
    --[no-]author <author>
                          override author for commit
    -m, --[no-]message <message>
                          commit message
    -u, --[no-]untracked-files[=<mode>]
                          show untracked files, optional modes: all, normal, no. (Default: all)
    --[no-]cleanup=<mode> how to strip spaces and #comments from message
`

// coreHelp is part of the output of core -h, from the cli package.
const coreHelp = `Command line tools for developing apps and libraries using the Cogent Core framework.

Usage:
    core [flags]
    core <subcommand> [flags]

Subcommands:
    build
        Build builds an executable for the package
        at the config path for the config platforms
    install

Flags:
  -v, -verbose bool
        whether to run the command in verbose mode
  -name string
        Name is the user-friendly name of the project.
`

// goHelp is part of the output of go -h.
const goHelp = `Go is a tool for managing Go source code.

Usage:

	go <command> [arguments]

The commands are:

	bug         start a bug report
	build       compile packages and dependencies

Use "go help <command>" for more information about a command.

Additional help topics:

	buildconstraint build constraints
	cache           build and test caching

Use "go help <topic>" for more information about that topic.
`

// gitHelpAll is part of the output of git help -a, with an example.
const gitHelpAll = `See 'git help <command>' to read about a specific subcommand

Main Porcelain Commands
   add                     Add file contents to the index
   commit                  Record changes to the repository

Ancillary Commands
   config                  Get and set repository or global options
   gc                      Cleanup unnecessary files and optimize the local repository

Low-level Commands / Manipulators
   apply                   Apply a patch to files and/or to the index
   checkout-index          Copy files from the index to the working tree

For example:
   ok      archive/tar   0.011s
`

// goModTidyHelp is part of the output of go help mod tidy, with a
// changed usage.
const goModTidyHelp = `usage: go mod tidy [-v] [-go=version] [-C dir] [-e | -x]

Tidy makes sure go.mod matches the source code in the module.

The -v flag causes tidy to print information about removed modules
to standard error.

The -go flag causes tidy to update the 'go' directive. More.
`

// gitStashHelp is part of the output of git stash -h.
const gitStashHelp = `usage: git stash list [<log-options>]
   or: git stash show [-u | --include-untracked] [<stash>]
   or: git stash [push] [-p | --patch]
   or: git stash save [-p | --patch] [<message>]

    -q, --[no-]quiet      be quiet
`

func TestSetFromHelp(t *testing.T) {
	cm := &SubCmd{Cmd: "git commit"}
	cm.SetFromHelp(gitCommitHelp)
	assert.Equal(t, "git commit [-a | --interactive | --patch] [-s] [-v] [-u[<mode>]] [--amend]\n[--dry-run] [(-c | -C | --squash) <commit> | --fixup [(amend|reword):]<commit>]", cm.Usage)
	require.Len(t, cm.Flags, 7)
	want := []Flag{
		{Name: "--quiet", Names: []string{"-q", "--quiet"}, Negatable: true, Doc: "suppress summary after successful commit"},
		{Name: "--verbose", Names: []string{"-v", "--verbose"}, Negatable: true, Doc: "show diff in commit message template"},
		{Name: "--file", Names: []string{"-F", "--file"}, Value: "<file>", Negatable: true, Doc: "read message from file"},
		{Name: "--author", Names: []string{"--author"}, Value: "<author>", Negatable: true, Doc: "override author for commit"},
		{Name: "--message", Names: []string{"-m", "--message"}, Value: "<message>", Negatable: true, Doc: "commit message"},
		{Name: "--untracked-files", Names: []string{"-u", "--untracked-files"}, Value: "<mode>", Optional: true, Negatable: true, Doc: "show untracked files, optional modes: all, normal, no. (Default: all)"},
		{Name: "--cleanup", Names: []string{"--cleanup"}, Value: "<mode>", Equals: true, Negatable: true, Doc: "how to strip spaces and #comments from message"},
	}
	for i, fl := range cm.Flags {
		assert.Equal(t, want[i], *fl)
	}
	assert.Empty(t, cm.Cmds)

	cm = &SubCmd{Cmd: "core"}
	cm.SetFromHelp(coreHelp)
	assert.Equal(t, "core [flags]\ncore <subcommand> [flags]", cm.Usage)
	require.Len(t, cm.Cmds, 2)
	assert.Equal(t, "core build", cm.Cmds[0].Cmd)
	assert.Equal(t, "Build builds an executable for the package at the config path for the config platforms", cm.Cmds[0].Doc)
	assert.Equal(t, "core install", cm.Cmds[1].Cmd)
	require.Len(t, cm.Flags, 2)
	assert.True(t, cm.Flags[0].IsBool(), "a bool value hint is a flag without a value")
	assert.Equal(t, "-verbose", cm.Flags[0].Name)
	assert.Equal(t, "string", cm.Flags[1].Value)

	// the help topics of go are not subcommands
	cm = &SubCmd{Cmd: "go"}
	cm.SetFromHelp(goHelp)
	require.Len(t, cm.Cmds, 2)
	assert.Equal(t, "go bug", cm.Cmds[0].Cmd)
	assert.Equal(t, "go build", cm.Cmds[1].Cmd)

	// the groups of git help -a, and an example in the text, which is not a
	// subcommand
	cm = &SubCmd{Cmd: "git"}
	cm.SetFromHelp(gitHelpAll)
	var groups []string
	for _, sub := range cm.Cmds {
		groups = append(groups, sub.Cmd+": "+sub.Group)
	}
	assert.Equal(t, []string{"git add: Main Porcelain Commands", "git commit: Main Porcelain Commands", "git config: Ancillary Commands", "git gc: Ancillary Commands"}, groups)

	// flags in the usage lines only, with their sentences in the text
	cm = &SubCmd{Cmd: "go mod tidy"}
	cm.SetFromHelp(goModTidyHelp)
	require.Len(t, cm.Flags, 5)
	assert.Equal(t, Flag{Name: "-v", Names: []string{"-v"}, Doc: "The -v flag causes tidy to print information about removed modules to standard error."}, *cm.Flags[0])
	assert.Equal(t, Flag{Name: "-go", Names: []string{"-go"}, Value: "version", Equals: true, Doc: "The -go flag causes tidy to update the 'go' directive."}, *cm.Flags[1])
	assert.Equal(t, "-C", cm.Flags[2].Name)
	assert.Equal(t, "dir", cm.Flags[2].Value)
	assert.Equal(t, "-e", cm.Flags[3].Name)
	assert.Equal(t, "-x", cm.Flags[4].Name)

	// long flags indented more than short ones, as of brew
	cm = &SubCmd{Cmd: "brew install"}
	cm.SetFromHelp("Usage: brew install [options] formula\n\n  -d, --debug                      If brewing fails, open an\n                                   debugging session.\n      --display-times              Print install times.\n  -f, --force                      Install formulae.\n")
	var names []string
	for _, fl := range cm.Flags {
		names = append(names, fl.Name+": "+fl.Doc)
	}
	assert.Equal(t, []string{"--debug: If brewing fails, open an debugging session.", "--display-times: Print install times.", "--force: Install formulae."}, names)

	// lists of subcommands separated by commas, as of npm -h, and flags
	// separated by |
	cm = &SubCmd{Cmd: "npm"}
	cm.SetFromHelp("npm <command>\n\nAll commands:\n\n    access, adduser, audit,\n    ci, help, install\n\nSpecify configs in the ini-formatted file:\n    /Users/x/.npmrc\n")
	var npm []string
	for _, sub := range cm.Cmds {
		npm = append(npm, sub.Cmd)
	}
	assert.Equal(t, []string{"npm access", "npm adduser", "npm audit", "npm ci", "npm install"}, npm)
	assert.Equal(t, "", cm.Cmds[0].Group)
	cm = &SubCmd{Cmd: "go build"}
	cm.SetFromHelp("The build flags are shared by the build, clean, and test commands:\n\n\t-C dir\n\t\tChange to dir before running the command, such as\n\t\tthe number of programs, such as build commands or\n")
	assert.Empty(t, cm.Cmds, "prose with commas")
	cm = &SubCmd{Cmd: "npm install"}
	cm.SetFromHelp("Install a package\n\n  -S|--save\n    Save installed packages.\n\n  --install-strategy <hoisted|nested>\n    Sets the strategy.\n")
	require.Len(t, cm.Flags, 2)
	assert.Equal(t, []string{"-S", "--save"}, cm.Flags[0].Names)
	assert.Equal(t, "<hoisted|nested>", cm.Flags[1].Value)

	// lists of subcommands under "==> " headings, as of brew commands,
	// without the developer commands
	cm = &SubCmd{Cmd: "brew"}
	cm.SetFromHelp("==> Built-in commands\n--cache\nanalytics\ninstall\n\n==> Built-in developer commands\naudit\nbottle\n")
	var listed []string
	for _, sub := range cm.Cmds {
		listed = append(listed, sub.Cmd+": "+sub.Group)
	}
	assert.Equal(t, []string{"brew analytics: Built-in commands", "brew install: Built-in commands"}, listed)
	assert.Equal(t, [][]string{{"brew", "-h"}, {"brew", "--help"}, {"brew", "help"}, {"brew", "help", "-a"}, {"man", "brew"}, {"brew", "commands"}}, helpCommands([]string{"brew"}))

	// subcommands in the usage lines only
	cm = &SubCmd{Cmd: "git stash"}
	cm.SetFromHelp(gitStashHelp)
	var subs []string
	for _, sub := range cm.Cmds {
		subs = append(subs, sub.Cmd)
	}
	assert.Equal(t, []string{"git stash list", "git stash show", "git stash save"}, subs)
	assert.Equal(t, "git stash list [<log-options>]", cm.Cmds[0].Doc)
	assert.Equal(t, "git stash list [<log-options>]\ngit stash show [-u | --include-untracked] [<stash>]\ngit stash [push] [-p | --patch]\ngit stash save [-p | --patch] [<message>]", cm.Usage)
}

func TestStripOverstrike(t *testing.T) {
	assert.Equal(t, "NAME bold", stripOverstrike("N\bNA\bAM\bME\bE b\bbo\bol\bld\bd"))
}

func TestCommandLine(t *testing.T) {
	cm := &SubCmd{Cmd: "git commit"}
	cm.SetFromHelp(gitCommitHelp)
	values := cm.FormStruct()
	v := reflect.ValueOf(values).Elem()
	assert.Equal(t, "git commit", cm.CommandLine(values))
	v.FieldByName("Arguments").SetString("a.go b.go")
	v.FieldByName("Verbose").SetBool(true)
	v.FieldByName("Message").SetString("fix the bug's cause")
	v.FieldByName("UntrackedFiles").SetString("no")
	v.FieldByName("Cleanup").SetString("strip")
	assert.Equal(t, `git commit --verbose --message 'fix the bug'\''s cause' --untracked-files=no --cleanup=strip a.go b.go`, cm.CommandLine(values))

	fl := &Flag{Name: "-u", Names: []string{"-u"}, Value: "<mode>", Optional: true}
	assert.Equal(t, "-uall", fl.Arg("all"), "an optional value is attached")
	field, _ := reflect.TypeOf(values).Elem().FieldByName("Message")
	assert.Equal(t, "-m, --message &lt;message&gt;", field.Tag.Get("label"), "shown as markup")
	assert.Equal(t, "commit message", field.Tag.Get("doc"))
}

// TestParse tests parsing the help of real commands, which are skipped if
// they are not installed.
func TestParse(t *testing.T) {
	for _, cmd := range []string{"git", "git commit", "go build", "ls"} {
		if _, err := exec.LookPath(cmd[:2]); err != nil {
			continue
		}
		cm, err := Parse(cmd)
		require.NoError(t, err, cmd)
		assert.NotEmpty(t, cm.Flags, cmd)
		assert.NotEmpty(t, cm.Doc, cmd)
	}
	if _, err := exec.LookPath("git"); err == nil {
		cm, err := ParseCmd("git")
		require.NoError(t, err)
		assert.True(t, cm.HasSubCmds())
		assert.NotNil(t, cm.SubCmd("commit"))
		assert.False(t, cm.SubCmd("commit").IsParsed())
	}
	if _, err := exec.LookPath("go"); err == nil {
		cm, err := ParseCmd("go")
		require.NoError(t, err)
		assert.NotNil(t, cm.SubCmd("build"))
		assert.Nil(t, cm.SubCmd("cache"), "a help topic")
	}
	if _, err := exec.LookPath("ls"); err == nil {
		cm, err := ParseCmd("ls")
		require.NoError(t, err)
		require.Len(t, cm.SubCmds, 1)
		assert.Equal(t, MainName, cm.SubCmds[0].Name)
		assert.True(t, cm.SubCmds[0].IsParsed())
	}
}

// testCommands returns commands saved in the given file, with git and its
// commit subcommand parsed, and ls, without running help commands.
func testCommands(t *testing.T, file string) *Commands {
	cs := &Commands{file: sharedfile.File[commandsData]{Path: file}, Parsed: map[string]*Cmd{}, Args: OpenArgs("")}
	cs.Entries = Standard()
	commit := &SubCmd{Name: "commit", Cmd: "git commit", Source: "git commit -h"}
	commit.SetFromHelp(gitCommitHelp)
	cs.Parsed["git"] = &Cmd{Cmd: "git", SubCmds: []*SubCmd{{Name: "status", Cmd: "git status", Doc: "Show the working tree status"}, commit}}
	cs.Parsed["ls"] = &Cmd{Cmd: "ls", SubCmds: []*SubCmd{{Name: MainName, Cmd: "ls", Source: "man ls", Flags: []*Flag{{Name: "-l", Names: []string{"-l"}}}}}}
	require.NoError(t, cs.Save())
	return cs
}

// TestCommands tests opening, saving, and caching the commands, which are
// shared by processes through their file.
func TestCommands(t *testing.T) {
	file := filepath.Join(t.TempDir(), "commands.json")
	cs := testCommands(t, file)
	_, err := os.Stat(file)
	require.NoError(t, err)

	// another process
	cs2 := &Commands{file: sharedfile.File[commandsData]{Path: file}, Parsed: map[string]*Cmd{}}
	cs2.reload()
	assert.Equal(t, version, cs2.Version)
	assert.Equal(t, "Git", cs2.Entries[0].Label())
	assert.Equal(t, "git", cs2.Entries[0].Command)
	got, err := cs2.SubCmd("git  commit", false) // cached, not parsed
	require.NoError(t, err)
	assert.Len(t, got.Flags, 7)
	ls, err := cs2.SubCmd("ls", false)
	require.NoError(t, err)
	assert.Equal(t, MainName, ls.Name)
	_, err = cs2.SubCmd("git", false)
	assert.Error(t, err, "git has subcommands")

	// the changes of each are read by the other, and merged
	time.Sleep(10 * time.Millisecond) // (a different modification time)
	require.NoError(t, cs2.update(func() {
		cs2.Parsed["git"].SubCmd("status").Source = "git status -h"
		cs2.Entries = cs2.Entries[1:]
	}))
	assert.True(t, cs.file.Changed())
	assert.NotNil(t, cs.cachedSub("git status"), "parsed by the other")
	assert.Equal(t, "go", cs.Entries[0].Command, "edited by the other")
	require.NoError(t, cs.update(func() {
		cs.Parsed["ls"].SubCmds[0].Doc = "list"
	}))
	assert.Equal(t, "list", cs2.cachedSub("ls").Doc)
	assert.NotNil(t, cs2.cachedSub("git status"), "kept")

	// a file of the first version has standard entries
	require.NoError(t, os.WriteFile(file, []byte(`{"Entries": [{"Name": "Git", "Children": [{"Command": "git status"}]}], "Parsed": {"git status": {"Cmd": "git status"}}}`), 0o644))
	cs3 := Open(file)
	assert.Equal(t, Standard()[0].Command, cs3.Entries[0].Command)
	assert.Nil(t, cs3.cached("git status"))
}

// TestSubCmdMenu tests that a parsed subcommand stays in its group, and the
// menu of the subcommands of git: the common ones, and then a submenu of
// the others.
func TestSubCmdMenu(t *testing.T) {
	cs := &Commands{Parsed: map[string]*Cmd{}, Entries: Standard()}
	cs.Parsed["git"] = &Cmd{Cmd: "git", SubCmds: []*SubCmd{
		{Name: "add", Cmd: "git add", Group: "Main"},
		{Name: "bisect", Cmd: "git bisect", Group: "Main"},
		{Name: "status", Cmd: "git status", Group: "Main"},
		{Name: "archive", Cmd: "git archive", Group: "Ancillary"},
	}}
	prev := parseSub
	parseSub = func(cmd string) (*SubCmd, error) { return &SubCmd{Cmd: cmd, Source: cmd + " -h"}, nil }
	t.Cleanup(func() { parseSub = prev })
	sc, err := cs.SubCmd("git bisect", false)
	require.NoError(t, err)
	assert.Equal(t, "Main", sc.Group)

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
	assert.Equal(t, []string{"Status", "Add", "More"}, texts(m))
	more := core.NewScene()
	m.Children[len(m.Children)-1].(*core.Button).Menu(more, image.Point{})
	assert.Equal(t, []string{"Archive", "Bisect"}, texts(more))

	cs.Entries[0].Common = "" // by group
	m = core.NewScene()
	cs.subCmdsMenu(m, "git", nil)
	assert.Equal(t, []string{"Add", "Bisect", "Status", "Ancillary"}, texts(m))
}

// TestArgs tests saving the arguments last used for a subcommand, and
// setting them in its form, in another process.
func TestArgs(t *testing.T) {
	file := filepath.Join(t.TempDir(), "commands-args.json")
	as := OpenArgs(file)
	sc := &SubCmd{Name: "commit", Cmd: "git commit"}
	sc.SetFromHelp(gitCommitHelp)
	values := sc.FormStruct()
	v := reflect.ValueOf(values).Elem()
	v.FieldByName("Arguments").SetString("a.go")
	v.FieldByName("Verbose").SetBool(true)
	v.FieldByName("Message").SetString("the message")
	require.NoError(t, as.Set(sc, values))
	saved := as.Cmds["git"].SubCmds["commit"]
	assert.Equal(t, map[string]string{"--verbose": "true", "--message": "the message"}, saved.Flags, "only those used")

	as2 := OpenArgs(file)
	values2 := sc.FormStruct()
	as2.SetValues(sc, values2)
	assert.Equal(t, sc.CommandLine(values), sc.CommandLine(values2))
	assert.Equal(t, `git commit --verbose --message 'the message' a.go`, sc.CommandLine(values2))
}

// TestDialog tests the dialog of the form of a command, and running its
// command line.
func TestDialog(t *testing.T) {
	imagex.UpdateTestImagesFor(t)
	cs := &Commands{Parsed: map[string]*Cmd{}, Args: OpenArgs("")}
	cm := &SubCmd{Name: "commit", Cmd: "git commit", Doc: "Record changes to the repository"}
	cm.SetFromHelp(gitCommitHelp)
	ran := ""
	d := cs.NewDialog(nil, cm, func(cmdline string) { ran = cmdline })
	d.AssertRender(t, "dialog", func() {
		fm := tree.ChildByType[*core.Form](d)
		require.NotNil(t, fm)
		v := reflect.ValueOf(fm.Struct).Elem()
		v.FieldByName("Message").SetString("the message")
		fm.Update()
		fm.SendChange()
	})
	assert.Equal(t, "", ran, "not run until Run is clicked")
}

// TestStandard tests the standard commands: those of this platform that
// are installed, and adding them to the entries of an earlier version.
func TestStandard(t *testing.T) {
	have := installed([]string{"ls", "no-such-command-xyz"})
	assert.True(t, have["ls"])
	assert.False(t, have["no-such-command-xyz"])

	ens := Standard()
	cmds := map[string]string{} // by category
	for _, en := range ens {
		for _, c := range en.Children {
			cmds[c.Command] = en.Name
		}
	}
	assert.Equal(t, "Files", cmds["ls"])
	assert.Equal(t, "Text", cmds["sed"])
	for _, c := range []string{"apt", "zypper", "winget"} {
		if runtime.GOOS != "linux" || c == "winget" {
			assert.NotContains(t, cmds, c, "not on this platform")
		}
	}
	if _, err := exec.LookPath("brew"); err == nil && runtime.GOOS == "darwin" {
		assert.Equal(t, "Packages", cmds["brew"])
	}

	old := []*Entry{{Command: "git"}, {Name: "Files", Children: []*Entry{{Command: "ls"}, {Command: "grep"}}}}
	merged := addStandard(old)
	assert.Equal(t, "git", merged[0].Command)
	assert.Equal(t, "Files", merged[1].Name)
	var files []string
	for _, c := range merged[1].Children {
		files = append(files, c.Command)
	}
	assert.Equal(t, []string{"ls", "grep", "find"}, files[:3], "kept, with the others added")
	n := 0
	for _, en := range merged {
		if en.Command == "git" {
			n++
		}
	}
	assert.Equal(t, 1, n, "no duplicates")
}

// TestBunLike tests a command whose subcommands print its help (as bun run
// -h does), whose subcommands are not taken as theirs, and that common
// subcommands that the command does not have (or a description typed as
// them) leave its menu as without them.
func TestBunLike(t *testing.T) {
	cs := &Commands{Parsed: map[string]*Cmd{}, Entries: []*Entry{{Command: "bun", Common: "bun js package manager"}}}
	help := "Usage: bun <command>\n\nCommands:\n  run       Execute a file\n  test      Run unit tests\n  install   Install dependencies\n"
	prev := parseSub
	parseSub = func(cmd string) (*SubCmd, error) {
		sc := &SubCmd{Cmd: cmd, Source: cmd + " -h"}
		sc.SetFromHelp(help)
		return sc, nil
	}
	t.Cleanup(func() { parseSub = prev })
	root := &SubCmd{Cmd: "bun"}
	root.SetFromHelp(help)
	cm := &Cmd{Cmd: "bun"}
	for _, sub := range root.Cmds {
		_, sub.Name = splitCmd(sub.Cmd)
		cm.SubCmds = append(cm.SubCmds, sub)
	}
	cs.Parsed["bun"] = cm
	sc, err := cs.SubCmd("bun run", false)
	require.NoError(t, err)
	assert.Empty(t, sc.Cmds, "those of bun")

	m := core.NewScene()
	cs.subCmdsMenu(m, "bun", nil)
	var texts []string
	for _, kid := range m.Children {
		if b, ok := kid.(*core.Button); ok {
			texts = append(texts, b.Text)
		}
	}
	assert.Equal(t, []string{"Run", "Test", "Install"}, texts)
}

// TestSharedFlags tests that the go subcommands whose usage names the build
// flags have those of go build, after their own, without those they have.
func TestSharedFlags(t *testing.T) {
	cs := &Commands{Parsed: map[string]*Cmd{}}
	cs.Parsed["go"] = &Cmd{Cmd: "go", SubCmds: []*SubCmd{{Name: "build", Cmd: "go build"}, {Name: "run", Cmd: "go run"}, {Name: "doc", Cmd: "go doc"}}}
	helps := map[string]string{
		"go build": "usage: go build [-o output] [build flags] [packages]\n\n\t-o output\n\t\twrite the output there\n\t-race\n\t\tenable data race detection\n\t-exec xprog\n\t\tthe build's exec\n",
		"go run":   "usage: go run [build flags] [-exec xprog] package [arguments...]\n\n\t-exec xprog\n\t\trun the program with xprog\n\t-n\n\t\tprint the commands\n",
		"go doc":   "usage: go doc [doc flags] [package|[package.]symbol]\n\n\t-all\n\t\tshow all the documentation\n\t-src\n\t\tshow the source\n",
	}
	prev := parseSub
	parseSub = func(cmd string) (*SubCmd, error) {
		sc := &SubCmd{Cmd: cmd, Source: cmd + " -h"}
		sc.SetFromHelp(helps[cmd])
		return sc, nil
	}
	t.Cleanup(func() { parseSub = prev })
	flags := func(sc *SubCmd) []string {
		var fs []string
		for _, fl := range sc.Flags {
			fs = append(fs, fl.Name+": "+fl.Doc)
		}
		return fs
	}
	run, err := cs.SubCmd("go run", false)
	require.NoError(t, err)
	assert.Equal(t, []string{"-exec: run the program with xprog", "-n: print the commands", "-o: write the output there", "-race: enable data race detection"}, flags(run))
	doc, err := cs.SubCmd("go doc", false)
	require.NoError(t, err)
	assert.Len(t, doc.Flags, 2, "no build flags")
	build, err := cs.SubCmd("go build", false)
	require.NoError(t, err)
	assert.Len(t, build.Flags, 3)
}

// TestMergeFlags tests merging the flags of other help, as of go help
// testflag for go test: after the flags, without those of their names.
func TestMergeFlags(t *testing.T) {
	flags := []*Flag{{Name: "-c", Names: []string{"-c"}}, {Name: "-v", Names: []string{"-v"}, Doc: "test"}}
	others := []*Flag{{Name: "-v", Names: []string{"-v"}, Doc: "build"}, {Name: "-run", Names: []string{"-run"}}, {Name: "-run", Names: []string{"-run"}}}
	var names []string
	for _, fl := range mergeFlags(flags, others) {
		names = append(names, fl.Name+":"+fl.Doc)
	}
	assert.Equal(t, []string{"-c:", "-v:test", "-run:"}, names)
	assert.Equal(t, [][]string{{"go", "help", "testflag"}}, extraFlagHelp["go test"])
}
