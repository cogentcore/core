// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package goal

// Run executes the given command string, waiting for the command to finish,
// handling the given arguments appropriately.
// If there is any error, it adds it to the goal, and triggers CancelExecution.
// It forwards output to [exec.Config.Stdout] and [exec.Config.Stderr] appropriately.
func (gl *Goal) Run(cmd any, args ...any) {
	gl.Exec(false, false, false, cmd, args...)
}

// RunErrOK executes the given command string, waiting for the command to finish,
// handling the given arguments appropriately.
// It does not stop execution if there is an error.
// If there is any error, it adds it to the goal. It forwards output to
// [exec.Config.Stdout] and [exec.Config.Stderr] appropriately.
func (gl *Goal) RunErrOK(cmd any, args ...any) {
	gl.Exec(true, false, false, cmd, args...)
}

// Start starts the given command string for running in the background,
// handling the given arguments appropriately.
// If there is any error, it adds it to the goal. It forwards output to
// [exec.Config.Stdout] and [exec.Config.Stderr] appropriately.
func (gl *Goal) Start(cmd any, args ...any) {
	gl.Exec(false, true, false, cmd, args...)
}

// Output executes the given command string, handling the given arguments
// appropriately. If there is any error, it adds it to the goal. It returns
// the stdout as a string and forwards stderr to [exec.Config.Stderr] appropriately.
func (gl *Goal) Output(cmd any, args ...any) string {
	return gl.Exec(false, false, true, cmd, args...)
}

// OutputErrOK executes the given command string, handling the given arguments
// appropriately. If there is any error, it adds it to the goal. It returns
// the stdout as a string and forwards stderr to [exec.Config.Stderr] appropriately.
func (gl *Goal) OutputErrOK(cmd any, args ...any) string {
	return gl.Exec(true, false, true, cmd, args...)
}
