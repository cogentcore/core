// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"fmt"
	"strings"
)

// Note: the maps in this file are derived from https://github.com/golang/go/blob/master/src/go/build/syslist.go

// Platform is a platform with an operating system and an architecture
type Platform struct {
	OS   string
	Arch string
}

// String returns the platform as a string in the form "os/arch"
func (p Platform) String() string {
	return p.OS + "/" + p.Arch
}

// OSSupported determines whether the given operating system is supported by GoKi. If it is, it returns nil.
// If it isn't, it returns an error detailing the issue with the operating system (not found or not supported).
func OSSupported(os string) error {
	supported, ok := SupportedOS[os]
	if !ok {
		return fmt.Errorf("could not find operating system %s; please check that you spelled it correctly", os)
	}
	if !supported {
		return fmt.Errorf("operating system %s exists but is not supported by GoKi", os)
	}
	return nil
}

// ArchSupported determines whether the given architecture is supported by GoKi. If it is, it returns nil.
// If it isn't, it also returns an error detailing the issue with the architecture (not found or not supported).
func ArchSupported(arch string) error {
	supported, ok := SupportedArch[arch]
	if !ok {
		return fmt.Errorf("could not find architecture %s; please check that you spelled it correctly", arch)
	}
	if !supported {
		return fmt.Errorf("architecture %s exists but is not supported by GoKi", arch)
	}
	return nil
}

// SetString sets the platform from the given string of format os[/arch]
func (p *Platform) SetString(platform string) error {
	before, after, found := strings.Cut(platform, "/")
	err := OSSupported(before)
	if err != nil {
		return fmt.Errorf("error parsing platform: %w", err)
	}
	if !found {
		*p = Platform{OS: before, Arch: "*"}
		return nil
	}
	err = ArchSupported(after)
	if err != nil {
		return fmt.Errorf("error parsing platform: %w", err)
	}
	*p = Platform{OS: before, Arch: after}
	return nil
}

func (p *Platform) UnmarshalJSON(b []byte) error {
	platform := string(b)
	platform = strings.ReplaceAll(platform, `"`, "") // the quotes get passed in
	return p.SetString(platform)
}

// ArchsForOS returns contains all of the architectures supported for
// each operating system.
var ArchsForOS = map[string][]string{
	"darwin":  {"386", "amd64", "arm", "arm64"},
	"windows": {"386", "amd64", "arm", "arm64"},
	"linux":   {"386", "amd64", "arm", "arm64"},
	"android": {"386", "amd64", "arm", "arm64"},
	"ios":     {"386", "amd64", "arm", "arm64"},
}

// SupportedOS is a map containing all operating systems and whether they are supported by GoKi.
var SupportedOS = map[string]bool{
	"aix":       false,
	"android":   true,
	"darwin":    true,
	"dragonfly": false,
	"freebsd":   false,
	"hurd":      false,
	"illumos":   false,
	"ios":       true,
	"js":        false,
	"linux":     true,
	"nacl":      false,
	"netbsd":    false,
	"openbsd":   false,
	"plan9":     false,
	"solaris":   false,
	"wasip1":    false,
	"windows":   true,
	"zos":       false,
}

// SupportedArch is a map containing all computer architectures and whether they are supported by GoKi.
var SupportedArch = map[string]bool{
	"386":         true,
	"amd64":       true,
	"amd64p32":    true,
	"arm":         true,
	"armbe":       true,
	"arm64":       true,
	"arm64be":     true,
	"loong64":     false,
	"mips":        false,
	"mipsle":      false,
	"mips64":      false,
	"mips64le":    false,
	"mips64p32":   false,
	"mips64p32le": false,
	"ppc":         false,
	"ppc64":       false,
	"ppc64le":     false,
	"riscv":       false,
	"riscv64":     false,
	"s390":        false,
	"s390x":       false,
	"sparc":       false,
	"sparc64":     false,
	"wasm":        false,
}
