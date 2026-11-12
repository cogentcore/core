// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mobile

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cogentcore.org/core/base/exec"
	"cogentcore.org/core/cmd/config"
)

// webgpuLibsIOS is the Go module of the libraries of WebGPU for iOS,
// which has those for the iOS simulator on arm64, selected with the
// iossimulator build tag, since its version of 2026-11-12.
const webgpuLibsIOS = "github.com/oliverbestmann/webgpu/libs-ios"

// checkSimulatorWGPU returns an error if the libraries of WebGPU for iOS
// used by the app in the current directory do not have the one for the
// iOS simulator on the given architecture (amd64 is always there).
func checkSimulatorWGPU(arch string) error {
	if arch != "arm64" {
		return nil
	}
	dir, err := exec.Output("go", "list", "-m", "-f", "{{.Dir}}", webgpuLibsIOS)
	dir = strings.TrimSpace(dir)
	if err != nil || dir == "" {
		return nil // (not used, or not downloaded yet: the build reports it)
	}
	if _, err := os.Stat(filepath.Join(dir, "arm64-simulator")); err != nil {
		return fmt.Errorf("the WebGPU libraries of the app (%s) have no library for the iOS simulator on arm64: update github.com/oliverbestmann/webgpu to a version after v1.37.2", webgpuLibsIOS)
	}
	return nil
}

// simDevice is an iOS simulator device, as listed by simctl.
type simDevice struct {
	UDID        string `json:"udid"`
	Name        string `json:"name"`
	State       string `json:"state"`
	IsAvailable bool   `json:"isAvailable"`
}

// simulatorDevice returns the UDID of the iOS simulator device to run the
// app on: a booted one, or else the last available iPhone of the newest
// runtime, which is booted, with the Simulator app opened to show it.
func simulatorDevice() (string, error) {
	out, err := exec.Output("xcrun", "simctl", "list", "devices", "available", "-j")
	if err != nil {
		return "", err
	}
	var list struct {
		Devices map[string][]simDevice `json:"devices"`
	}
	if err := json.Unmarshal([]byte(out), &list); err != nil {
		return "", err
	}
	var runtimes []string
	for rt, devs := range list.Devices {
		for _, d := range devs {
			if d.State == "Booted" {
				return d.UDID, nil
			}
		}
		if strings.Contains(rt, ".iOS-") {
			runtimes = append(runtimes, rt)
		}
	}
	best, bestRT := "", ""
	for _, rt := range runtimes {
		for _, d := range list.Devices[rt] {
			if strings.HasPrefix(d.Name, "iPhone") && runtimeNewer(rt, bestRT) {
				best, bestRT = d.UDID, rt
			}
		}
	}
	if best == "" {
		return "", errors.New("no available iPhone simulator; add one in Xcode")
	}
	if err := exec.Run("xcrun", "simctl", "boot", best); err != nil {
		return "", err
	}
	exec.Run("open", "-a", "Simulator")
	return best, nil
}

// runtimeNewer returns whether the simulator runtime identifier a (as
// com.apple.CoreSimulator.SimRuntime.iOS-26-2) is a newer version than b,
// which is newer than none ("").
func runtimeNewer(a, b string) bool {
	if b == "" {
		return true
	}
	va, vb := runtimeVersion(a), runtimeVersion(b)
	for i := range min(len(va), len(vb)) {
		if va[i] != vb[i] {
			return va[i] > vb[i]
		}
	}
	return len(va) > len(vb)
}

// runtimeVersion returns the numbers of the version of the given
// simulator runtime identifier.
func runtimeVersion(rt string) []int {
	_, v, _ := strings.Cut(rt, ".iOS-")
	var nums []int
	for s := range strings.SplitSeq(v, "-") {
		var n int
		fmt.Sscan(s, &n)
		nums = append(nums, n)
	}
	return nums
}

// RunSimulator installs the app, which has been built for the iOS
// simulator, on a simulator device (see [simulatorDevice]) and launches
// it, showing its output if debugging.
func RunSimulator(c *config.Config) error {
	dev, err := simulatorDevice()
	if err != nil {
		return err
	}
	if err := exec.Run("xcrun", "simctl", "install", dev, filepath.Join(c.Build.Output, c.Name+".app")); err != nil {
		return fmt.Errorf("error installing app: %w", err)
	}
	args := []string{"simctl", "launch", "--terminate-running-process"}
	if c.Build.Debug {
		args = append(args, "--console-pty")
	}
	args = append(args, dev, c.ID)
	return exec.Verbose().SetBuffer(false).Run("xcrun", args...)
}
