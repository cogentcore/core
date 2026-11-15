//go:build !windows

package cmd

func windowsRegistryAddPath(_ string) error {
	return nil // no-op
}
