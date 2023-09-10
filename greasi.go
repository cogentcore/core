// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package greasi

import (
	"fmt"

	"goki.dev/grease"
)

// Run runs the given app with the given default
// configuration file paths. It is similar to
// [grease.Run], but it also runs the GUI if no
// arguments were provided. The app should be
// a pointer, and configuration options should
// be defined as fields on the app type. Also,
// commands should be defined as methods on the
// app type with the suffix "Cmd"; for example,
// for a command named "build", there should be
// the method:
//
//	func (a *App) BuildCmd() error
//
// Run uses [os.Args] for its arguments.
func Run(app, cfg any) error {
	leftovers, err := grease.Config(cfg)
	if err != nil {
		return fmt.Errorf("error configuring app: %w", err)
	}
	if len(leftovers) == 0 {
		GUI(app, cfg)
		return nil
	}
	cmd := leftovers[0]

	err = grease.RunCmd(app, cfg, cmd)
	if err != nil {
		return fmt.Errorf("error running command %q: %w", cmd, err)
	}
	return nil
}
