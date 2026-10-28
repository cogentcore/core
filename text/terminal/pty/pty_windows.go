// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build windows

package pty

import (
	"context"
	"os/exec"

	"github.com/charmbracelet/x/xpty"
)

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
