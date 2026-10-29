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

func start(cmd *exec.Cmd, cols, rows int) (pseudoTerminal, error) {
	return nil, ErrUnsupported
}

func wait(ctx context.Context, cmd *exec.Cmd) error {
	return ErrUnsupported
}

func open(cols, rows int) (pseudoTerminal, *os.File, error) {
	return nil, nil, ErrUnsupported
}

func signals(pt pseudoTerminal) bool {
	return false
}
