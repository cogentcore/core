// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package packman

import (
	"errors"
	"fmt"

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
		err := xe.Run(xe.VerboseConfig(), "adb", "logcat", "-c")
		if err != nil {
			return fmt.Errorf("error clearing logs: %w", err)
		}
	}
	err := xe.Run(xe.VerboseConfig(), "adb", "logcat", "*:"+c.Log.All, "Go:I", "GoLog:I")
	if err != nil {
		return fmt.Errorf("erroring getting logs: %w", err)
	}
	return nil
}
