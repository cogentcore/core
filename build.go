package packman

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
)

// Build builds an executable for the package at the given path for the given platforms
func Build(pkgPath string, platforms ...Platform) error {
	if len(platforms) == 0 {
		return errors.New("build: expected at least 1 platform")
	}
	err := os.MkdirAll("./bin/build", 0700)
	if err != nil {
		return fmt.Errorf("build: failed to create bin/build directory: %w", err)
	}
	androidArchs := []string{}
	for _, platform := range platforms {
		supported, ok := SupportedOS[platform.OS]
		if !supported {
			return fmt.Errorf("build: operating system %s exists but is not supported by GoKi", platform.OS)
		}
		if !ok {
			return fmt.Errorf("build: could not find operating system %s; please check that you spelled it correctly", platform.OS)
		}
		if platform.Arch != "all" {
			supported, ok = SupportedArch[platform.Arch]
			if !supported {
				return fmt.Errorf("build: architecture %s exists but is not supported by GoKi", platform.Arch)
			}
			if !ok {
				return fmt.Errorf("build: could not find architecture %s; please check that you spelled it correctly", platform.Arch)
			}
		}
		if platform.OS == "android" {
			androidArchs = append(androidArchs, platform.Arch)
			continue
		}
		if platform.OS == "ios" {
			// TODO: implement ios
			continue
		}
		if platform.OS == "js" {
			// TODO: implement js
			continue
		}
		err := buildDesktop(pkgPath, platform)
		if err != nil {
			return err
		}
	}
	if len(androidArchs) != 0 {
		return buildMobile(pkgPath, "android", androidArchs)
	}
	return nil
}

// buildDesktop builds an executable for the package at the given path for the given desktop platform.
// buildDesktop does not check whether platforms are valid, so it should be called through Build in almost all cases.
func buildDesktop(pkgPath string, platform Platform) error {
	cmd := exec.Command("go", "build", "-o", "./bin/build/", pkgPath)
	cmd.Env = []string{"GOOS=" + platform.OS, "GOARCH=" + platform.Arch}
	fmt.Println(cmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error building for platform %s/%s: %s", platform.OS, platform.Arch, string(output))
	}
	fmt.Println(string(output))
	return nil
}

// buildMobile builds an executable for the package at the given path for the given operating system and architectures.
// buildMobile does not check whether operating systems and architectures are valid, so it should be called through Build in almost all cases.
func buildMobile(pkgPath string, os string, archs []string) error {
	target := ""
	for i, arch := range archs {
		target += os + "/" + arch
		if i != len(archs)-1 {
			target += ","
		}
	}
	cmd := exec.Command("gomobile", "build", "-o", "./bin/build", "-target", target, pkgPath)
	return cmd.Run()
}
