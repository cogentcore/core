// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cogentcore.org/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setArgs sets [os.Args] for a test, restoring them after it.
func setArgs(t *testing.T, args ...string) {
	t.Helper()
	old := os.Args
	os.Args = append([]string{"myapp"}, args...)
	t.Cleanup(func() { os.Args = old })
}

// testOptions returns options for tests, which return errors instead
// of exiting, and print nothing on success.
func testOptions() *Options {
	opts := DefaultOptions("My App", "My App does things")
	opts.Fatal = false
	opts.PrintSuccess = false
	opts.DefaultFiles = nil
	return opts
}

type posConfig struct {
	// the name of the thing
	Name string `posarg:"0"`

	// the optional kind
	Kind string `posarg:"1" required:"-"`

	// the rest
	Rest []string `posarg:"leftover"`

	// how many
	Count int `default:"3"`

	// build options
	Build buildConfig `cmd:"pos-build,build"`

	called string
}

type buildConfig struct {
	// the target to build for
	Target string `default:"linux"`
}

func (pc *posConfig) OnConfig(cmd string) error {
	if pc.Name == "bad" {
		return errors.New("bad name")
	}
	return nil
}

func posBuild(pc *posConfig) error {
	pc.called = "build"
	return nil
}

func posFail(pc *posConfig) error { return errors.New("it failed") }

func init() {
	types.AddFunc(&types.Func{
		Name:       types.FuncName(posBuild),
		Doc:        "posBuild builds the thing.\n\nMore details here.",
		Directives: []types.Directive{{Tool: "cli", Directive: "cmd", Args: []string{"-root"}}},
	})
	types.AddFunc(&types.Func{
		Name:       types.FuncName(posFail),
		Directives: []types.Directive{{Tool: "cli", Directive: "bad"}},
	})
}

// TestCmdFromFunc tests making commands from functions, with their
// documentation and directives from types.
func TestCmdFromFunc(t *testing.T) {
	c, err := cmdFromFunc(posBuild)
	require.NoError(t, err)
	assert.Equal(t, "pos-build", c.Name)
	assert.True(t, c.Root, "from the directive")
	assert.True(t, strings.HasPrefix(c.Doc, "Pos build builds the thing."), c.Doc)

	_, err = cmdFromFunc(posFail)
	assert.ErrorContains(t, err, "unrecognized comment directive")

	cs, err := CmdsFromCmdOrFuncs[*posConfig]([]func(*posConfig) error{posBuild, posFail})
	assert.Error(t, err)
	assert.Nil(t, cs)

	cs, err = CmdsFromCmdOrFuncs[*posConfig]([]*Cmd[*posConfig]{{Name: "a"}})
	require.NoError(t, err)
	assert.Equal(t, "a", cs[0].Name)
}

// TestAddCmd tests adding commands, which keeps a single root.
func TestAddCmd(t *testing.T) {
	cmds := []*Cmd[int]{{Name: "a", Root: true}}
	cmds = AddCmd(cmds, &Cmd[int]{Name: "a"})
	assert.Len(t, cmds, 1, "same name")
	b := &Cmd[int]{Name: "b", Root: true}
	cmds = AddCmd(cmds, b)
	assert.Len(t, cmds, 2)
	assert.False(t, b.Root, "there is already a root")
	cmds = AddCmd([]*Cmd[int]{}, &Cmd[int]{Name: "c", Root: true})
	assert.True(t, cmds[0].Root)
}

// TestParseDirective tests parsing comment directives.
func TestParseDirective(t *testing.T) {
	d, err := ParseDirective("//cli:cmd -root -name 'my cmd'")
	require.NoError(t, err)
	require.NotNil(t, d)
	assert.Equal(t, "cli", d.Tool)
	assert.Equal(t, "cmd", d.Directive)
	assert.Equal(t, []string{"-root", "-name", "my cmd"}, d.Args)

	d, err = ParseDirective("types:add")
	require.NoError(t, err)
	assert.Equal(t, "add", d.Directive)
	assert.Nil(t, d.Args)

	for _, c := range []string{"", "// cli:cmd", "//cli:cmd\nmore", "//no directive"} {
		d, err = ParseDirective(c)
		assert.NoError(t, err)
		assert.Nil(t, d, c)
	}
	_, err = ParseDirective(`//cli:cmd "unclosed`)
	assert.Error(t, err)
}

// TestRunFuncs tests running commands given as functions, with
// positional arguments, command fields, and errors.
func TestRunFuncs(t *testing.T) {
	cfg := &posConfig{}
	setArgs(t, "pos-build", "thing", "-target", "js")
	require.NoError(t, Run(testOptions(), cfg, posBuild))
	assert.Equal(t, "build", cfg.called)
	assert.Equal(t, "thing", cfg.Name)
	assert.Equal(t, "js", cfg.Build.Target)
	assert.Equal(t, 3, cfg.Count)

	cfg = &posConfig{}
	setArgs(t, "thing", "kind", "x", "y")
	require.NoError(t, Run(testOptions(), cfg, posBuild), "the root")
	assert.Equal(t, "kind", cfg.Kind)
	assert.Equal(t, []string{"x", "y"}, cfg.Rest)

	setArgs(t, "pos-fail", "thing")
	err := Run(testOptions(), &posConfig{}, &Cmd[*posConfig]{Name: "pos-fail", Func: posFail}, &Cmd[*posConfig]{Name: "other", Func: posBuild})
	assert.ErrorContains(t, err, "it failed")

	setArgs(t, "-count", "x", "thing")
	assert.Error(t, Run(testOptions(), &posConfig{}, posBuild), "a bad flag value")

	setArgs(t, "bad")
	assert.ErrorContains(t, Run(testOptions(), &posConfig{}, posBuild), "bad name", "from OnConfig")

	setArgs(t)
	assert.ErrorContains(t, Run(testOptions(), &posConfig{}, posBuild), "missing positional argument 0")

	setArgs(t, "-name", "thing")
	assert.NoError(t, Run(testOptions(), &posConfig{}, posBuild), "a positional argument as a flag")

	_, err = CmdsFromCmdOrFuncs[*posConfig]([]func(*posConfig) error{posFail})
	require.Error(t, err)
	assert.ErrorContains(t, Run(testOptions(), &posConfig{}, posFail), "internal/programmer error")

	assert.ErrorContains(t, runCmd(testOptions(), &posConfig{}, "nope", &Cmd[*posConfig]{Name: "a"}), `command "nope" not found`)
	assert.Equal(t, "myapp", cmdString(""))
	assert.Equal(t, "myapp build", cmdString("build"))
}

// TestConfigFileFlag tests a config file given with the config flag,
// and searching up for include paths.
func TestConfigFileFlag(t *testing.T) {
	dir := t.TempDir()
	fn := filepath.Join(dir, "my.toml")
	require.NoError(t, os.WriteFile(fn, []byte("Count = 7\n"), 0o644))
	opts := testOptions()
	opts.SearchUp = true
	cfg := &posConfig{}
	setArgs(t, "-config", fn, "thing")
	cmd, err := config(opts, cfg)
	require.NoError(t, err)
	assert.Equal(t, "", cmd)
	assert.Equal(t, 7, cfg.Count)
	assert.Contains(t, opts.IncludePaths, "/", "searched up to the root")

	opts = testOptions()
	opts.NeedConfigFile = true
	opts.DefaultFiles = []string{"missing.toml"}
	_, err = config(opts, &posConfig{})
	assert.ErrorContains(t, err, "no config files found")
}

// TestUsageText tests the usage text of the root and of a command.
func TestUsageText(t *testing.T) {
	cmds := []*Cmd[*posConfig]{
		{Name: "build", Doc: "Build builds the thing.\n\nIt does it well.", Root: true},
		{Name: "mod", Doc: "Mod manages modules.\n\nModules have files."},
		{Name: "mod init", Doc: "Init makes a module."},
		{Name: "mod tidy"},
	}
	us := usage(testOptions(), &posConfig{}, "", cmds...)
	for _, s := range []string{"My App does things", "Usage:", "<name>", "[kind]", "Default command:", "Build builds the thing.", "Subcommands:", "help", "Mod manages modules.", "-count", "(default: 3)", "-config"} {
		assert.Contains(t, us, s)
	}
	assert.NotContains(t, us, "Modules have files.", "only the first paragraph")
	assert.NotContains(t, us, "Init makes a module.", "not one deeper")

	us = usage(testOptions(), &posConfig{}, "mod", cmds...)
	assert.Contains(t, us, "Mod manages modules.")
	assert.Contains(t, us, "Init makes a module.")
	assert.Contains(t, us, "mod tidy")
	assert.Contains(t, us, "<subcommand>")

	us = usage(testOptions(), &posConfig{}, "build", cmds...)
	assert.Contains(t, us, "-target", "the flags of the command")
	assert.NotContains(t, us, "Subcommands:")
}
