package packman

import (
	"fmt"
	"os"
	"path/filepath"
)

// AppName returns the app name for the package at the given path
func AppName(pkgPath string) string {
	if base := filepath.Base(filepath.Dir(pkgPath)); base != "." {
		return base
	}
	dir, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Unable to determine app name from package path and current working directory; please set it in your configuration file or as an argument to this command. (Could not get current working directory:", err.Error()+")")
		return "GoKi"
	}
	return filepath.Base(dir)
}

// BuildPath returns the path to the build binary output directory from the given package path.
func BuildPath(pkgPath string) string {
	return filepath.Join(pkgPath, "bin", "build")
}
