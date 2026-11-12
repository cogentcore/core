// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mobile

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"cogentcore.org/core/base/exec"
	"cogentcore.org/core/base/logx"
	"cogentcore.org/core/cmd/config"
)

// webgpuModule is the Go module of WebGPU, whose libraries of wgpu-native
// for iOS are built for devices only, not for the simulator.
const webgpuModule = "github.com/oliverbestmann/webgpu"

// wgpuVersionRE matches the version of wgpu-native in the update-wgpu.sh
// script of the [webgpuModule], from which its libraries are.
var wgpuVersionRE = regexp.MustCompile(`(?m)^VERSION="(v[0-9.]+)"`)

// simulatorWGPU returns the directory of the static library of wgpu-native
// for the iOS simulator on the given architecture, of the version of the
// libraries of the [webgpuModule] used by the app in the current directory.
// It is downloaded from the release of wgpu-native, once, into the cache
// directory of the user. It is put first in the library search path of the
// linker (CGO_LDFLAGS comes before the flags of the packages), instead of
// that of the library for devices.
func simulatorWGPU(arch string) (string, error) {
	dir, err := exec.Output("go", "list", "-m", "-f", "{{.Dir}}", webgpuModule)
	if err != nil || strings.TrimSpace(dir) == "" {
		return "", fmt.Errorf("finding the module %s of the app: %w", webgpuModule, err)
	}
	script, err := os.ReadFile(filepath.Join(strings.TrimSpace(dir), "update-wgpu.sh"))
	if err != nil {
		return "", fmt.Errorf("finding the version of wgpu-native: %w", err)
	}
	m := wgpuVersionRE.FindSubmatch(script)
	if m == nil {
		return "", errors.New("finding the version of wgpu-native: no VERSION in update-wgpu.sh")
	}
	version := string(m[1])
	rarch := map[string]string{"arm64": "aarch64", "amd64": "x86_64"}[arch]
	if rarch == "" {
		return "", fmt.Errorf("no wgpu-native for the iOS simulator on %s", arch)
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	name := "wgpu-ios-" + rarch + "-simulator-release"
	libDir := filepath.Join(cache, "cogentcore", "wgpu-native", version, name)
	lib := filepath.Join(libDir, "libwgpu_native.a")
	if _, err := os.Stat(lib); err == nil {
		return libDir, nil
	}
	url := "https://github.com/gfx-rs/wgpu-native/releases/download/" + version + "/" + name + ".zip"
	logx.PrintlnInfo("downloading", url)
	if err := downloadZipFile(url, "lib/libwgpu_native.a", lib); err != nil {
		return "", fmt.Errorf("downloading wgpu-native for the iOS simulator: %w", err)
	}
	return libDir, nil
}

// downloadZipFile downloads the zip file at the given URL and extracts
// the file of the given name in it to the given path.
func downloadZipFile(url, name, path string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: %s", url, resp.Status)
	}
	tmp, err := os.CreateTemp("", "wgpu-*.zip")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()
	size, err := io.Copy(tmp, resp.Body)
	if err != nil {
		return err
	}
	zr, err := zip.NewReader(tmp, size)
	if err != nil {
		return err
	}
	f, err := zr.Open(name)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	// written to a temporary file first, so that a partial one is not used
	out, err := os.CreateTemp(filepath.Dir(path), ".lib-*")
	if err != nil {
		return err
	}
	defer os.Remove(out.Name())
	if _, err := io.Copy(out, f); err != nil {
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Rename(out.Name(), path)
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
