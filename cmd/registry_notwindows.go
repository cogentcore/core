//go:build !windows

package cmd

func windowsRegistryAddPath(path string) error {
	return nil // no-op
}
