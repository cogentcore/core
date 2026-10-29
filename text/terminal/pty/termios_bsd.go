// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build darwin || dragonfly || freebsd || netbsd || openbsd

package pty

import "golang.org/x/sys/unix"

const ioctlGetTermios = unix.TIOCGETA
