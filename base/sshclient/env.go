// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sshclient

import (
	"bytes"
	"maps"
	"regexp"
	"slices"
	"strings"
	"time"
)

// loginEnvCmd prints the environment of a command on the host, and then,
// after a marker, that of an interactive login shell of the user, after
// another marker, such that output of its startup files comes before it.
// The variables are separated by NUL (GNU env -0), or else by newlines.
const loginEnvCmd = `env -0 2>/dev/null || env; printf '\001END\001'; ` +
	`"$SHELL" -lic 'printf "\001ENV\001"; env -0 2>/dev/null || env' </dev/null`

// loginEnvSkip are the variables of the login shell that are not set for
// the commands: those of the shell itself, its terminal and the session.
var loginEnvSkip = map[string]bool{
	"_": true, "SHLVL": true, "PWD": true, "OLDPWD": true, "PS1": true, "PS2": true,
	"PS4": true, "PROMPT_COMMAND": true, "TERM": true, "COLUMNS": true, "LINES": true,
	"SSH_CLIENT": true, "SSH_CONNECTION": true, "SSH_TTY": true,
}

// envName matches the name of an environment variable.
var envName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// getLoginEnv gets the environment variables that an interactive login
// shell of the user on the host sets, as in its ~/.bashrc after the check
// that the shell is interactive, such as PATH (see [Config.LoginEnv]),
// which are then set for each command, as [Client.exports].
func (cl *Client) getLoginEnv() {
	ses, err := cl.NewSession(false)
	if err != nil {
		return
	}
	defer ses.Close()
	timeout := cl.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	stop := time.AfterFunc(timeout, func() { ses.Close() }) // as for a .bashrc that waits
	defer stop.Stop()
	var buf bytes.Buffer
	ses.Stdout, ses.Stderr = &buf, nil
	ses.Run(loginEnvCmd) // the exit status is that of the shell
	plain, login, ok := bytes.Cut(buf.Bytes(), []byte("\001END\001"))
	if !ok {
		return
	}
	i := bytes.LastIndex(login, []byte("\001ENV\001"))
	if i < 0 {
		return
	}
	cl.exports = loginExports(parseEnv(plain), parseEnv(login[i+len("\001ENV\001"):]))
}

// parseEnv returns the variables of the given output of env, separated by
// NUL, or else by newlines (when values with newlines are lost).
func parseEnv(out []byte) map[string]string {
	sep := []byte("\n")
	if bytes.IndexByte(out, 0) >= 0 {
		sep = []byte{0}
		if j := bytes.LastIndexByte(out, 0); j >= 0 {
			out = out[:j] // anything after the last one is not of env
		}
	}
	env := map[string]string{}
	for _, kv := range bytes.Split(out, sep) {
		k, v, ok := strings.Cut(string(kv), "=")
		if ok && envName.MatchString(k) {
			env[k] = v
		}
	}
	return env
}

// loginExports returns the shell code that sets the variables of the
// given login environment that differ from those of the given plain one,
// to start a command with, or "" if there are none.
func loginExports(plain, login map[string]string) string {
	var b strings.Builder
	for _, k := range slices.Sorted(maps.Keys(login)) {
		v := login[k]
		if loginEnvSkip[k] || strings.HasPrefix(k, "BASH_FUNC_") || plain[k] == v {
			continue
		}
		if b.Len() == 0 {
			b.WriteString("export")
		}
		b.WriteString(" " + k + "='" + strings.ReplaceAll(v, "'", `'\''`) + "'")
	}
	if b.Len() == 0 {
		return ""
	}
	return b.String() + "; "
}
