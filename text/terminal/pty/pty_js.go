// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package pty

import (
	"context"
	"os"
	"os/exec"
)

func start(_ *exec.Cmd, _, _ int) (pseudoTerminal, error) {
	return nil, ErrUnsupported
}

func wait(_ context.Context, _ *exec.Cmd) error {
	return ErrUnsupported
}

func open(_, _ int) (pseudoTerminal, *os.File, *os.File, error) {
	return nil, nil, nil, ErrUnsupported
}

func signals(_ pseudoTerminal) bool {
	return false
}
