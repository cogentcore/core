package packman

import (
	"fmt"
	"os/exec"
	"runtime"
)

// Install installs the package with the given ID by looking for it in the list of supported packages
func Install(pkgID string) error {
	packages, err := LoadPackages()
	if err != nil {
		return fmt.Errorf("error loading packages: %w", err)
	}
	for _, pkg := range packages {
		if pkg.ID == pkgID {
			return InstallPackage(pkg)
		}
	}
	return fmt.Errorf("error: could not find package %s", pkgID)
}

// InstallPackage installs the given package
func InstallPackage(pkg Package) error {
	fmt.Println("Installing", pkg.Name)
	commands, ok := pkg.InstallCommands[runtime.GOOS]
	if !ok {
		return fmt.Errorf("error: the requested package (%s) does not support your operating system (%s)", pkg.Name, runtime.GOOS)
	}
	for _, command := range commands {
		cmd := exec.Command(command.Name, command.Args...)
		b, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("error installing %s: %w", pkg.Name, err)
		}
		fmt.Println(string(b))
	}
	fmt.Println("Successfully installed", pkg.Name)
	return nil
}
