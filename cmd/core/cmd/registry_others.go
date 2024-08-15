//go:build !windows

package cmd

func WindowsRegistryAddPath(path string) error {
	return nil
}
