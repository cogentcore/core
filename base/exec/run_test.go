// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package exec

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cogentcore.org/core/base/goscript"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// quiet returns a config whose output, errors, and echo go to buffers.
func quiet() (c *Config, out, errs, echo *bytes.Buffer) {
	out, errs, echo = &bytes.Buffer{}, &bytes.Buffer{}, &bytes.Buffer{}
	c = &Config{Buffer: true, Env: map[string]string{}, Echo: echo}
	c.SetStdout(out).SetStderr(errs).SetStdin(strings.NewReader(""))
	return
}

// TestOutputEnvDir tests running commands with output, environment
// variables (expanded in the arguments too), and a directory.
func TestOutputEnvDir(t *testing.T) {
	goscript.Skip(t, "runs commands")
	c, out, errs, echo := quiet()
	c.SetEnv("CORE_EXEC_TEST", "value")
	s, err := c.Output("sh", "-c", "echo $CORE_EXEC_TEST; echo err >&2")
	require.NoError(t, err)
	assert.Equal(t, "value", s, "the variable in the environment")
	assert.Contains(t, out.String(), "value", "and in the output")
	assert.Contains(t, errs.String(), "err")
	assert.Contains(t, echo.String(), "sh -c", "the command is echoed")

	args := []string{"$CORE_EXEC_TEST"}
	s, err = c.Output("echo", args...)
	require.NoError(t, err)
	assert.Equal(t, "value", s, "expanded in the arguments")

	dir := t.TempDir()
	c.Dir = dir
	s, err = c.Output("pwd")
	require.NoError(t, err)
	real, _ := filepath.EvalSymlinks(dir)
	rs, _ := filepath.EvalSymlinks(s)
	assert.Equal(t, real, rs)
	assert.Contains(t, echo.String(), dir, "the directory is echoed")
}

// TestErrors tests the errors and exit codes of commands.
func TestErrors(t *testing.T) {
	goscript.Skip(t, "runs commands")
	c, _, _, _ := quiet()
	err := c.Run("sh", "-c", "exit 3")
	require.Error(t, err)
	var re *RunError
	require.True(t, errors.As(err, &re))
	assert.Contains(t, re.Error(), "exit 3")
	assert.Equal(t, 3, ExitStatus(errors.Unwrap(err)))
	assert.True(t, CmdRan(errors.Unwrap(err)), "it ran, and failed")
	assert.Equal(t, 0, ExitStatus(nil))
	assert.True(t, CmdRan(nil))
	assert.Equal(t, 1, ExitStatus(errors.New("other")))
	assert.False(t, CmdRan(errors.New("other")))

	err = c.Run("core-exec-no-such-command")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "command not found")
}

// TestStartAndPrintOnly tests starting commands without waiting, and
// printing commands without running them.
func TestStartAndPrintOnly(t *testing.T) {
	goscript.Skip(t, "runs commands")
	c, out, _, _ := quiet()
	cmd, err := c.Start("sh", "-c", "echo started")
	require.NoError(t, err)
	require.NoError(t, cmd.Wait())
	assert.Contains(t, out.String(), "started", "a started command is not buffered")

	c, out, _, echo := quiet()
	c.PrintOnly = true
	dir := filepath.Join(t.TempDir(), "made")
	require.NoError(t, c.Run("sh", "-c", "echo ran"))
	assert.Empty(t, out.String(), "not run")
	require.NoError(t, c.MkdirAll(dir, 0755))
	assert.NoDirExists(t, dir)
	assert.Contains(t, echo.String(), "mkdir -p")

	c.PrintOnly = false
	require.NoError(t, c.MkdirAll(dir, 0755))
	assert.DirExists(t, dir)
	require.NoError(t, c.RemoveAll(dir))
	assert.NoDirExists(t, dir)
	assert.Contains(t, echo.String(), "rm -rf")
	p, err := LookPath("sh")
	assert.NoError(t, err)
	assert.NotEmpty(t, p)
}

// TestCmdIO tests running commands with their own standard IO state.
func TestCmdIO(t *testing.T) {
	goscript.Skip(t, "runs commands")
	c, _, _, _ := quiet()
	cio := NewCmdIO(c)
	assert.Equal(t, "<nil>", cio.String())
	var ob bytes.Buffer
	cio.Out = &ob
	require.NoError(t, c.RunIO(cio, "echo", "one"))
	assert.Contains(t, ob.String(), "one")
	assert.Contains(t, cio.String(), "exit status 0")

	s, err := c.OutputIO(cio, "echo", "two")
	require.NoError(t, err)
	assert.Equal(t, "two", s)
	assert.Contains(t, ob.String(), "two", "also written to the output")

	require.NoError(t, c.StartIO(cio, "sh", "-c", "exit 0"))
	assert.NotContains(t, cio.String(), "no process info", "a started process has a pid")
	require.NoError(t, cio.Cmd.Wait())
	cio.Cmd = &Cmd{}
	assert.Contains(t, cio.String(), "no process info")
}

// TestConfigs tests the standard configs and setting them.
func TestConfigs(t *testing.T) {
	goscript.Skip(t, "runs commands")
	for _, get := range []func() *Config{Major, Minor, Verbose, Silent} {
		c := get()
		assert.True(t, c.Buffer)
		assert.NotNil(t, c.Env)
	}
	c, out, _, _ := quiet()
	defer SetMajor(nil)
	defer SetMinor(nil)
	defer SetVerbose(nil)
	defer SetSilent(nil)
	SetMajor(c)
	SetMinor(c)
	SetVerbose(c)
	SetSilent(c)
	for _, get := range []func() *Config{Major, Minor, Verbose, Silent} {
		g := get()
		assert.NotSame(t, c, g, "a copy")
		assert.Equal(t, out, g.StdIO.Out)
	}

	require.NoError(t, Run("echo", "major run"))
	s, err := Output("echo", "major output")
	require.NoError(t, err)
	assert.Equal(t, "major output", s)
	cmd, err := Start("sh", "-c", "exit 0")
	require.NoError(t, err)
	require.NoError(t, cmd.Wait())
	assert.Contains(t, out.String(), "major run")

	dir := filepath.Join(t.TempDir(), "d")
	require.NoError(t, MkdirAll(dir, 0755))
	require.NoError(t, RemoveAll(dir))
	PrintCmd("printed", nil)
	assert.Contains(t, c.Echo.(*bytes.Buffer).String(), "printed")

	var w bytes.Buffer
	assert.Nil(t, c.GetWriter(nil, nil))
	assert.Equal(t, &w, c.GetWriter(&w, nil))
	assert.Equal(t, c.StdIO.Err, c.GetWriter(nil, errors.New("e")), "the error writer for an error")
	assert.NoError(t, os.Setenv("CORE_EXEC_TEST_OS", "os"))
	s, err = c.Output("echo", "$CORE_EXEC_TEST_OS")
	require.NoError(t, err)
	assert.Equal(t, "os", s, "from the environment of the process")
}

// TestStdIOPrintWrap tests printing to standard IO, and wrapped IO.
func TestStdIOPrintWrap(t *testing.T) {
	var out, errs bytes.Buffer
	st := &StdIO{Out: &out, Err: &errs}
	st.Print("a", 1)
	st.Println("b")
	st.Printf("c%d", 2)
	st.ErrPrint("x")
	st.ErrPrintln("y")
	st.ErrPrintf("z%d", 3)
	assert.Equal(t, "a1b\nc2", out.String())
	assert.Equal(t, "xy\nz3", errs.String())

	var o2 StdIO
	prev := o2.Set(st)
	assert.Equal(t, StdIO{}, *prev)
	assert.Equal(t, st.Out, o2.Out)

	var w StdIO
	assert.Nil(t, w.GetWrapped())
	assert.Nil(t, w.SetWrappers(nil))
	w.NewWrappers(st)
	assert.Nil(t, (&StdIO{Out: &out}).SetWrappers(st), "not wrapped")
	got := w.GetWrapped()
	assert.Equal(t, st.Out, got.Out)
	var o3, e3 bytes.Buffer
	r3 := strings.NewReader("in")
	prevw := w.SetWrappers(&StdIO{Out: &o3, Err: &e3, In: r3})
	assert.Equal(t, st.Out, prevw.Out)
	w.Print("wrapped")
	assert.Equal(t, "wrapped", o3.String())
	w.SetWrappedOut(&out)
	w.SetWrappedErr(&errs)
	w.SetWrappedIn(strings.NewReader("x"))
	w.Print("!")
	assert.True(t, strings.HasSuffix(out.String(), "!"))

	ss := &StdIOState{}
	ss.Out = &out
	assert.True(t, ss.ErrIsInOut(&out))
	assert.False(t, ss.ErrIsInOut(&errs))
	ss.PushOut(&errs)
	assert.True(t, ss.ErrIsInOut(&out), "in the stack")
	ss.PopOut()
}

// TestConfigCopiesEnv tests that the copies of the set default configs
// have their own environment variables.
func TestConfigCopiesEnv(t *testing.T) {
	sets := []func(c *Config){SetMajor, SetMinor, SetVerbose, SetSilent}
	gets := []func() *Config{Major, Minor, Verbose, Silent}
	for i, set := range sets {
		set(&Config{Env: map[string]string{}})
		a := gets[i]()
		a.SetEnv("CORE_EXEC_TEST_COPY", "a")
		_, has := gets[i]().Env["CORE_EXEC_TEST_COPY"]
		assert.False(t, has, "the environment of a copy is not shared: %d", i)
		set(nil)
	}
}

// TestExpandArgsCopy tests that expanding the variables of the args does
// not change the caller's args.
func TestExpandArgsCopy(t *testing.T) {
	goscript.Skip(t, "runs commands")
	c := Silent().SetEnv("CORE_EXEC_TEST_VAR", "expanded")
	args := []string{"$CORE_EXEC_TEST_VAR"}
	out, err := c.Output("echo", args...)
	assert.NoError(t, err)
	assert.Equal(t, "expanded", out)
	assert.Equal(t, []string{"$CORE_EXEC_TEST_VAR"}, args, "the args are not changed")
}
