package packman

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

// OSSupported determines whether the given operating system is supported by GoKi. If it is, it returns nil.
// If it isn't, it returns an error detailing the issue with the operating system (not found or not supported).
func OSSupported(os string) error {
	supported, ok := SupportedOS[os]
	if !supported {
		return fmt.Errorf("operating system %s exists but is not supported by GoKi", os)
	}
	if !ok {
		return fmt.Errorf("could not find operating system %s; please check that you spelled it correctly", os)
	}
	return nil
}

// ArchSupported determines whether the given architecture is supported by GoKi. If it is, it returns nil.
// If it isn't, it also returns an error detailing the issue with the architecture (not found or not supported).
func ArchSupported(arch string) error {
	supported, ok := SupportedArch[arch]
	if !supported {
		return fmt.Errorf("architecture %s exists but is not supported by GoKi", arch)
	}
	if !ok {
		return fmt.Errorf("could not find architecture %s; please check that you spelled it correctly", arch)
	}
	return nil
}

// ParsePlatform parses the given platform string of format os[/arch]
func ParsePlatform(platform string) (Platform, error) {
	before, after, found := strings.Cut(platform, "/")
	err := OSSupported(before)
	if err != nil {
		return Platform{}, fmt.Errorf("error parsing platform: %w", err)
	}
	if !found {
		return Platform{OS: before, Arch: "all"}, nil
	}
	err = ArchSupported(after)
	if err != nil {
		return Platform{}, fmt.Errorf("error parsing platform: %w", err)
	}
	return Platform{OS: before, Arch: after}, nil
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
