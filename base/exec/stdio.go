// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package exec

import (
	"io"
	"os"

	"cogentcore.org/core/base/stack"
)

// StdIO manages standard input / output Reader / Writers and
// stacks thereof.
type StdIO struct {
	// Out is the writer to write the standard output of called commands to.
	// It can be set to nil to disable the writing of the standard output.
	Out io.Writer

	// Err is the writer to write the standard error of called commands to.
	// It can be set to nil to disable the writing of the standard error.
	Err io.Writer

	// In is the reader to use as the standard input.
	In io.Reader

	// OutStack is stack of out
	OutStack stack.Stack[io.Writer]

	// ErrStack is stack of err
	ErrStack stack.Stack[io.Writer]

	// InStack is stack of in
	InStack stack.Stack[io.Reader]

	// Starting depths of the respective stacks, for unwinding the stack
	// to a defined starting point.
	OutStart, ErrStart, InStart int
}

// StdAll sets all to os.Std*
func (st *StdIO) StdAll() {
	st.Out = os.Stdout
	st.Err = os.Stderr
	st.In = os.Stdin
}

// PushOut pushes the new io.Writer as the current
// Stdout, saving the previous one on a stack.
// Use Popout to restore previous.
func (st *StdIO) PushOut(out io.Writer) {
	st.OutStack.Push(st.Out)
	st.Out = out
}

// PopOut restores previous io.Writer as Stdout
// from the stack, saved during PushOut,
// returning the one that was previously current.
func (st *StdIO) PopOut() io.Writer {
	cur := st.Out
	st.Out = st.OutStack.Pop()
	return cur
}

// PushErr pushes the new io.Writer as the current
// Stderr, saving the previous one on a stack.
// Use PopErr to restore previous.
func (st *StdIO) PushErr(err io.Writer) {
	st.ErrStack.Push(st.Err)
	st.Err = err
}

// PopErr restores previous io.Writer as Stderr
// from the stack, saved during PushErr,
// returning the one that was previously current.
func (st *StdIO) PopErr() io.Writer {
	cur := st.Err
	st.Err = st.ErrStack.Pop()
	return cur
}

// PushIn pushes the new io.Reader as the current
// Stdin, saving the previous one on a stack.
// Use Popin to restore previous.
func (st *StdIO) PushIn(in io.Reader) {
	st.InStack.Push(st.In)
	st.In = in
}

// PopIn restores previous io.Reader as Stdin
// from the stack, saved during Pushin,
// returning the one that was previously current.
func (st *StdIO) PopIn() io.Reader {
	cur := st.In
	st.In = st.InStack.Pop()
	return cur
}

// StackStart records the starting depths of the Std stacks
func (st *StdIO) StackStart() {
	st.OutStart = len(st.OutStack)
	st.ErrStart = len(st.ErrStack)
	st.InStart = len(st.InStack)
}

// PopToStart unwinds the Std stacks to the depths recorded at StackStart.
// if closeErr is true, close the error file -- otherwise not
// (typically a copy of out)
func (st *StdIO) PopToStart(closeErr bool) {
	for len(st.OutStack) > st.OutStart {
		CloseWriter(st.PopOut())
	}
	for len(st.ErrStack) > st.ErrStart {
		if closeErr {
			CloseWriter(st.PopErr())
		} else {
			st.PopErr()
		}
	}
	for len(st.InStack) > st.InStart {
		CloseReader(st.PopIn())
	}
}

// CloseWriter closes given Writer, if it has a Close() method
func CloseWriter(w io.Writer) {
	if st, ok := w.(io.WriteCloser); ok {
		st.Close()
	}
}

// CloseReader closes given Reader, if it has a Close() method
func CloseReader(r io.Reader) {
	if st, ok := r.(io.ReadCloser); ok {
		st.Close()
	}
}
