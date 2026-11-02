// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !((darwin && !ios) || (linux && !android) || freebsd || openbsd || netbsd || dragonfly)

package apps

import "errors"

func forFile(file string) ([]App, error) {
	return nil, errors.ErrUnsupported
}

func openWith(file string, app App) error {
	return errors.ErrUnsupported
}

func setDefault(file string, app App) error {
	return errors.ErrUnsupported
}

func allApps() ([]App, error) {
	return nil, errors.ErrUnsupported
}
