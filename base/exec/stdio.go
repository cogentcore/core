// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package exec

import (
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"

	"cogentcore.org/core/base/stack"
)

// StdIO contains one set of standard input / output Reader / Writers.
type StdIO struct {
	// Out is the writer to write the standard output of called commands to.
	// It can be set to nil to disable the writing of the standard output.
	Out io.Writer

	// Err is the writer to write the standard error of called commands to.
	// It can be set to nil to disable the writing of the standard error.
	Err io.Writer

	// In is the reader to use as the standard input.
	In io.Reader
}

// SetFromOS sets all our IO to current os.Std*
func (st *StdIO) SetFromOS() {
	st.Out, st.Err, st.In = os.Stdout, os.Stderr, os.Stdin
}

// SetAll sets all our IO from given args
func (st *StdIO) SetAll(out, err io.Writer, in io.Reader) {
	st.Out, st.Err, st.In = out, err, in
}

// Set sets our values from other StdIO, returning
// the current values at time of call, to restore later.
func (st *StdIO) Set(o *StdIO) *StdIO {
	cur := *st
	*st = *o
	return &cur
}

// SetToOS sets the current IO to os.Std*, returning
// a StdIO with the current IO settings prior to this call,
// which can be used to restore.
// Note: os.Std* are *os.File types, and this function will panic
// if the current IO are not actually *os.Files.
// The results of a prior SetToOS call will do the right thing for
// saving and restoring the os state.
func (st *StdIO) SetToOS() *StdIO {
	cur := &StdIO{}
	cur.SetFromOS()
	os.Stdout = st.Out.(*os.File)
	os.Stderr = st.Err.(*os.File)
	os.Stdin = st.In.(*os.File)
	return cur
}

// Print prints to the [StdIO.Out] Stdout
func (st *StdIO) Print(v ...any) {
	fmt.Fprint(st.Out, v...)
}

// Println prints to the [StdIO.Out]
func (st *StdIO) Println(v ...any) {
	fmt.Fprintln(st.Out, v...)
}

// Printf prints to the [StdIO.Out] Stdout
func (st *StdIO) Printf(f string, v ...any) {
	fmt.Fprintf(st.Out, f, v...)
}

// IsPipe returns true if the given object is an os.File corresponding to a Pipe,
// which is also not the same as the current os.Stdout, in case that is a Pipe.
func IsPipe(rw any) bool {
	if rw == nil {
		return false
	}
	w, ok := rw.(io.Writer)
	if !ok {
		return false
	}
	if w == os.Stdout {
		return false
	}
	of, ok := rw.(*os.File)
	if !ok {
		return false
	}
	st, err := of.Stat()
	if err != nil {
		return false
	}
	md := st.Mode()
	if md&fs.ModeNamedPipe != 0 {
		return true
	}
	return md&fs.ModeCharDevice == 0
}

// OutIsPipe returns true if current Out is a Pipe
func (st *StdIO) OutIsPipe() bool { return IsPipe(st.Out) }

// StdIOState maintains a stack of StdIO settings for easier management
// of dynamic IO routing.  Call [StackStart] prior to
// setting the IO values using Push commands, and then call
// [PopToStart] when done to close any open IO and reset.
type StdIOState struct {
	StdIO

	// OutStack is stack of out
	OutStack stack.Stack[io.Writer]

	// ErrStack is stack of err
	ErrStack stack.Stack[io.Writer]

	// InStack is stack of in
	InStack stack.Stack[io.Reader]

	// PipeIn is a stack of the os.File to use for reading from the Out,
	// when Out is a Pipe, created by [PushOutPipe].
	// Use [OutIsPipe] function to determine if the current output is a Pipe
	// in order to determine whether to use the current [PipeIn.Peek()].
	// These will be automatically closed during [PopToStart] whenever the
	// corresponding Out is a Pipe.
	PipeIn stack.Stack[*os.File]

	// Starting depths of the respective stacks, for unwinding the stack
	// to a defined starting point.
	OutStart, ErrStart, InStart int
}

// PushOut pushes a new io.Writer as the current [Out],
// saving the current one on a stack, which can be restored using [PopOut].
func (st *StdIOState) PushOut(out io.Writer) {
	st.OutStack.Push(st.Out)
	st.Out = out
}

// PushOutPipe calls os.Pipe() and pushes the writer side
// as the new [Out], and pushes the Reader side to [PipeIn]
// which should then be used as the [In] for any other relevant process.
// Call [OutIsPipe] to determine if the current Out is a Pipe, to know
// whether to use the PipeIn.Peek() value.
func (st *StdIOState) PushOutPipe() {
	r, w, err := os.Pipe()
	if err != nil {
		slog.Error(err.Error())
	}
	st.PushOut(w)
	st.PipeIn.Push(r)
}

// PopOut restores previous io.Writer as [Out] from the stack,
// saved during [PushOut], returning the current Out at time of call.
// Pops and closes corresponding PipeIn if current Out is a Pipe.
// This does NOT close the current one, in case you need to use it before closing,
// so that is your responsibility (see [PopToStart] that does this for you).
func (st *StdIOState) PopOut() io.Writer {
	if st.OutIsPipe() && len(st.PipeIn) > 0 {
		CloseReader(st.PipeIn.Pop())
	}
	cur := st.Out
	st.Out = st.OutStack.Pop()
	return cur
}

// PushErr pushes a new io.Writer as the current [Err],
// saving the current one on a stack, which can be restored using [PopErr].
func (st *StdIOState) PushErr(err io.Writer) {
	st.ErrStack.Push(st.Err)
	st.Err = err
}

// PopErr restores previous io.Writer as [Err] from the stack,
// saved during [PushErr], returning the current Err at time of call.
// This does NOT close the current one, in case you need to use it before closing,
// so that is your responsibility (see [PopToStart] that does this for you).
// Note that Err is often the same as Out, in which case only Out should be closed.
func (st *StdIOState) PopErr() io.Writer {
	cur := st.Err
	st.Err = st.ErrStack.Pop()
	return cur
}

// PushIn pushes a new [io.Reader] as the current [In],
// saving the current one on a stack, which can be restored using [PopIn].
func (st *StdIOState) PushIn(in io.Reader) {
	st.InStack.Push(st.In)
	st.In = in
}

// PopIn restores previous io.Reader as [In] from the stack,
// saved during [PushIn], returning the current In at time of call.
// This does NOT close the current one, in case you need to use it before closing,
// so that is your responsibility (see [PopToStart] that does this for you).
func (st *StdIOState) PopIn() io.Reader {
	cur := st.In
	st.In = st.InStack.Pop()
	return cur
}

// StackStart records the starting depths of the IO stacks
func (st *StdIOState) StackStart() {
	st.OutStart = len(st.OutStack)
	st.ErrStart = len(st.ErrStack)
	st.InStart = len(st.InStack)
}

// PopToStart unwinds the IO stacks to the depths recorded at [StackStart],
// automatically closing the ones that are popped.
// It automatically checks if any of the Err items are the same as Out
// and does not redundantly close those.
func (st *StdIOState) PopToStart() {
	for len(st.ErrStack) > st.ErrStart {
		er := st.PopErr()
		if !st.ErrIsInOut(er) {
			fmt.Println("close err")
			CloseWriter(er)
		}
	}
	for len(st.OutStack) > st.OutStart {
		CloseWriter(st.PopOut())
	}
	for len(st.InStack) > st.InStart {
		st.PopIn()
	}
}

// ErrIsInOut returns true if the given Err writer is also present
// within the active (> [OutStart]) stack of Outs.
// If this is true, then Err should not be closed, as it will be closed
// when the Out is closed.
func (st *StdIOState) ErrIsInOut(er io.Writer) bool {
	for i := st.OutStart; i < len(st.OutStack); i++ {
		if st.OutStack[i] == er {
			return true
		}
	}
	return false
}

// CloseWriter closes given Writer, if it has a Close() method
func CloseWriter(w io.Writer) {
	if st, ok := w.(io.Closer); ok {
		st.Close()
	}
}

// CloseReader closes given Reader, if it has a Close() method
func CloseReader(r io.Reader) {
	if st, ok := r.(io.Closer); ok {
		st.Close()
	}
}

// WriteWrapper is an io.Writer that wraps another io.Writer
type WriteWrapper struct {
	io.Writer
}

// ReadWrapper is an io.Reader that wraps another io.Reader
type ReadWrapper struct {
	io.Reader
}

// NewWrappers initializes this StdIO with wrappers around given StdIO
func (st *StdIO) NewWrappers(o *StdIO) {
	st.Out = &WriteWrapper{Writer: o.Out}
	st.Err = &WriteWrapper{Writer: o.Err}
	st.In = &ReadWrapper{Reader: o.In}
}

// SetWrappers sets the wrappers to current values from given StdIO,
// returning a copy of the wrapped values previously in place at start of call,
// which can be used in restoring state later.
// The wrappers must have been created using NewWrappers initially.
func (st *StdIO) SetWrappers(o *StdIO) *StdIO {
	cur := st.GetWrapped()
	st.Out.(*WriteWrapper).Writer = o.Out
	st.Err.(*WriteWrapper).Writer = o.Err
	st.In.(*ReadWrapper).Reader = o.In
	return cur
}

// SetWrappedOut sets the wrapped Out to given writer.
// The wrappers must have been created using NewWrappers initially.
func (st *StdIO) SetWrappedOut(w io.Writer) {
	st.Out.(*WriteWrapper).Writer = w
}

// SetWrappedErr sets the wrapped Err to given writer.
// The wrappers must have been created using NewWrappers initially.
func (st *StdIO) SetWrappedErr(w io.Writer) {
	st.Err.(*WriteWrapper).Writer = w
}

// SetWrappedIn sets the wrapped In to given reader.
// The wrappers must have been created using NewWrappers initially.
func (st *StdIO) SetWrappedIn(r io.Reader) {
	st.In.(*ReadWrapper).Reader = r
}

// GetWrapped returns the current wrapped values as a StdIO.
// The wrappers must have been created using NewWrappers initially.
func (st *StdIO) GetWrapped() *StdIO {
	o := &StdIO{}
	o.Out = st.Out.(*WriteWrapper).Writer
	o.Err = st.Err.(*WriteWrapper).Writer
	o.In = st.In.(*ReadWrapper).Reader
	return o
}
