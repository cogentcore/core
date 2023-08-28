// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package packman

import (
	"bufio"
	"errors"
	"fmt"
	"os/exec"

	"goki.dev/goki/config"
)

// Log prints the logs from your app running on the
// config operating system (android or ios) to the terminal
func Log(c *config.Config) error {
	if c.Log.Target == "ios" {
		return errors.New("ios not supported yet")
	}
	if !c.Log.Keep {
		cmd := exec.Command("adb", "logcat", "-c")
		fmt.Println(CmdString(cmd))
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("error clearing logs: %w, %s", err, string(output))
		}
		fmt.Println(string(output))
	}
	cmd := exec.Command("adb", "logcat", "*:"+c.Log.All, "Go:I", "GoLog:I")
	fmt.Println(CmdString(cmd))
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("erroring getting logs: %w", err)
	}
	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("error starting logging: %w", err)
	}
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		t := scanner.Text()
		fmt.Println(t)
	}
	err = cmd.Wait()
	if err != nil {
		return fmt.Errorf("error logging: %w", err)
	}
	return nil
}
