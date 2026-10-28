// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package pty

import (
	"context"
	"errors"
	"os/exec"
)

// ErrUnsupported is returned on platforms without pseudo-terminals.
var ErrUnsupported = errors.New("pty: pseudo-terminals are not supported on this platform")

func start(cmd *exec.Cmd, cols, rows int) (pseudoTerminal, error) {
	return nil, ErrUnsupported
}

func wait(ctx context.Context, cmd *exec.Cmd) error {
	return ErrUnsupported
}
