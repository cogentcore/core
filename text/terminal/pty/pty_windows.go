// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build windows

package pty

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/x/xpty"
	"golang.org/x/sys/windows"
)

var (
	kernel32                  = windows.NewLazySystemDLL("kernel32.dll")
	procAttachConsole         = kernel32.NewProc("AttachConsole")
	procFreeConsole           = kernel32.NewProc("FreeConsole")
	procSetConsoleCtrlHandler = kernel32.NewProc("SetConsoleCtrlHandler")

	// attached is whether this process is attached to the console of
	// a pseudo-terminal of [Run].
	attached atomic.Bool

	// ctrlHandler is a console control handler that ignores Control+C
	// and Control+Break in this process.
	ctrlHandler = sync.OnceValue(func() uintptr {
		return windows.NewCallback(func(event uint32) uintptr {
			if event == windows.CTRL_C_EVENT || event == windows.CTRL_BREAK_EVENT {
				return 1
			}
			return 0
		})
	})
)

// defaultInputMode is the mode of the input of a new console.
const defaultInputMode = windows.ENABLE_PROCESSED_INPUT | windows.ENABLE_LINE_INPUT |
	windows.ENABLE_ECHO_INPUT | windows.ENABLE_MOUSE_INPUT | windows.ENABLE_INSERT_MODE |
	windows.ENABLE_QUICK_EDIT_MODE | windows.ENABLE_EXTENDED_FLAGS | windows.ENABLE_AUTO_POSITION

func start(cmd *exec.Cmd, cols, rows int) (pseudoTerminal, error) {
	pt, err := xpty.NewConPty(cols, rows)
	if err != nil {
		return nil, err
	}
	if err := pt.Start(cmd); err != nil {
		pt.Close()
		return nil, err
	}
	return pt, nil
}

func wait(ctx context.Context, cmd *exec.Cmd) error {
	return xpty.WaitProcess(ctx, cmd)
}

// conPty is a pseudo-console that this process is attached to, for [Run].
type conPty struct {
	*xpty.ConPty

	// in is the input of the console.
	in *os.File

	// std are the standard handles of this process before it was attached,
	// which are those of the console while it is, for the code that uses
	// them, such as to set their modes.
	std []windows.Handle

	detachOnce sync.Once
}

// stdHandles are the identifiers of the standard handles of a process.
var stdHandles = [3]uint32{windows.STD_INPUT_HANDLE, windows.STD_OUTPUT_HANDLE, windows.STD_ERROR_HANDLE}

// open opens a new pseudo-console and attaches this process to it, as
// only the processes in a console can open its input and output, which
// it returns.
func open(cols, rows int) (pseudoTerminal, *os.File, *os.File, error) {
	if !attached.CompareAndSwap(false, true) {
		return nil, nil, nil, errors.New("pty: on Windows, only one function can run with a pseudo-terminal at a time")
	}
	cp, err := xpty.NewConPty(cols, rows)
	if err != nil {
		attached.Store(false)
		return nil, nil, nil, err
	}
	pt := &conPty{ConPty: cp}
	out, err := pt.attach()
	if err != nil {
		pt.Close()
		return nil, nil, nil, err
	}
	return pt, pt.in, out, nil
}

// attach attaches this process to the pseudo-console, through a helper
// process in it that waits for input, which is ended once this process
// is attached, before any input is sent.
func (pt *conPty) attach() (*os.File, error) {
	comspec := os.Getenv("ComSpec")
	if comspec == "" {
		comspec = "cmd.exe"
	}
	helper := exec.Command(comspec, "/d", "/c", "pause>nul")
	if err := pt.Start(helper); err != nil {
		return nil, err
	}
	procFreeConsole.Call() // the console of this process, if any
	var err error
	for start := time.Now(); ; time.Sleep(10 * time.Millisecond) { // until the helper is in it
		r, _, e := procAttachConsole.Call(uintptr(helper.Process.Pid))
		if r != 0 {
			break
		}
		if time.Since(start) > 5*time.Second {
			err = fmt.Errorf("pty: attaching to the console: %w", e)
			break
		}
	}
	helper.Process.Kill()
	helper.Process.Wait()
	if err != nil {
		return nil, err
	}
	procSetConsoleCtrlHandler.Call(0, 0) // the commands get Control+C, even if this process ignored it
	procSetConsoleCtrlHandler.Call(ctrlHandler(), 1)
	if pt.in, err = openConsole("CONIN$"); err != nil {
		return nil, err
	}
	out, err := openConsole("CONOUT$")
	if err != nil {
		pt.in.Close()
		return nil, err
	}
	// the helper ended without restoring the mode it set
	windows.SetConsoleMode(windows.Handle(pt.in.Fd()), defaultInputMode)
	for i, h := range []*os.File{pt.in, out, out} {
		std, _ := windows.GetStdHandle(stdHandles[i])
		pt.std = append(pt.std, std)
		windows.SetStdHandle(stdHandles[i], windows.Handle(h.Fd()))
	}
	return out, nil
}

// openConsole opens the given input or output of the console.
func openConsole(name string) (*os.File, error) {
	h, err := windows.CreateFile(windows.StringToUTF16Ptr(name), windows.GENERIC_READ|windows.GENERIC_WRITE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("pty: opening %s: %w", name, err)
	}
	return os.NewFile(uintptr(h), name), nil
}

// detach detaches this process from the pseudo-console.
func (pt *conPty) detach() {
	pt.detachOnce.Do(func() {
		for i, h := range pt.std {
			windows.SetStdHandle(stdHandles[i], h)
		}
		procSetConsoleCtrlHandler.Call(ctrlHandler(), 0)
		procFreeConsole.Call()
		attached.Store(false)
	})
}

// Close detaches this process from the pseudo-console before closing
// it, which would otherwise end this process.
func (pt *conPty) Close() error {
	pt.detach()
	return pt.ConPty.Close()
}

func signals(pt pseudoTerminal) bool {
	p, ok := pt.(*conPty)
	if !ok || p.in == nil {
		return false
	}
	var mode uint32
	return windows.GetConsoleMode(windows.Handle(p.in.Fd()), &mode) == nil && mode&windows.ENABLE_PROCESSED_INPUT != 0
}
