// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package packman

import (
	"errors"
	"fmt"
	"os"

	"goki.dev/goki/config"
	"goki.dev/xe"
)

// Log prints the logs from your app running on the
// config operating system (android or ios) to the terminal.
//
//gti:add
func Log(c *config.Config) error {
	if c.Log.Target == "ios" {
		return errors.New("ios not supported yet")
	}
	if !c.Log.Keep {
		err := xe.Run("adb", "logcat", "-c")
		if err != nil {
			return fmt.Errorf("error clearing logs: %w", err)
		}
	}
	// we are logging continiously so we can't buffer, and we must forcefully pipe stdout and stderr
	err := xe.Major().SetBuffer(false).SetStdout(os.Stdout).SetStderr(os.Stderr).Run("adb", "logcat", "*:"+c.Log.All, "Go:D", "GoLog:D")
	if err != nil {
		return fmt.Errorf("erroring getting logs: %w", err)
	}
	return nil
}
