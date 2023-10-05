// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tools

import (
	"archive/tar"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"goki.dev/goki/config"
	"goki.dev/xe"
)

// Setup does platform-specific setup that ensures that development can be done
// for the config platform, mostly by installing necessary tools.
//
//gti:add
func Setup(c *config.Config) error {
	switch c.Setup.Platform.OS {
	// TODO: support more platforms in setup
	case "ios":
		return SetupIOS(c)
	}
	return nil
}

// SetupIOS is the implementation of [Setup] for iOS.
func SetupIOS(c *config.Config) error {
	murl := "https://github.com/KhronosGroup/MoltenVK/releases/latest/download/MoltenVK-ios.tar"
	lname := "MoltenVK/MoltenVK/dylib/iOS/libMoltenVK.dylib"
	hdir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("error getting user home directory: %w", err)
	}
	gdir := filepath.Join(hdir, "Library", "goki")
	tlfname := "_tmp_goki_setup_libMoltenVK.dylib"
	tlpath := filepath.Join(gdir, tlfname)

	resp, err := http.Get(murl)
	if err != nil {
		return fmt.Errorf("error downloading iOS framework tar from latest MoltenVK release at url %q: %w", murl, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("got status code %d when downloading iOS framework tar from latest MoltenVK release at url %q", resp.StatusCode, murl)
	}
	tr := tar.NewReader(resp.Body)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break // end of tar data
		}
		if err != nil {
			return fmt.Errorf("error reading iOS framework tar: %w", err)
		}
		if hdr.Name != lname {
			continue
		}
		err = xe.MkdirAll(gdir, 0750)
		if err != nil {
			return fmt.Errorf("error creating directory for MoltenVK dylib: %w", err)
		}
		f, err := os.Create(tlpath)
		if err != nil {
			return fmt.Errorf("error creating file for MoltenVK dylib: %w", err)
		}
		defer f.Close()
		_, err = io.Copy(f, tr)
		if err != nil {
			return fmt.Errorf("error copying MoltenVK dylib to MoltenVK dylib file: %w", err)
		}
		err = xe.Major().SetDir(gdir).Run("lipo", "-create", tlfname, "-output", "MoltenVK")
		if err != nil {
			return fmt.Errorf("error creating library from MoltenVK dylib file: %w", err)
		}
		err = os.Remove(tlpath)
		if err != nil {
			return fmt.Errorf("error removing temporary MoltenVK dylib file: %w", err)
		}

		err = xe.MkdirAll(gdir+"/MoltenVK.framework", 0750)
		if err != nil {
			return fmt.Errorf("error making directory for MoltenK framework: %w", err)
		}
		err = os.Rename(gdir+"/MoltenVK", gdir+"/MoltenVK.framework/MoltenVK")
		if err != nil {
			return fmt.Errorf("error moving MoltenVK library into MoltenVK framework: %w", err)
		}
		err = xe.Major().SetDir(gdir+"/MoltenVK.framework").Run("install_name_tool", "-change", "@rpath/libMoltenVK.dylib", "@executable_path/MoltenVK.framework/MoltenVK", "MoltenVK")
		if err != nil {
			return fmt.Errorf("error changing executable path of MoltenVK library inside of MoltenVK framework: %w", err)
		}
		return nil
	}
	return fmt.Errorf("internal error: did not find MoltenVK dylib at %q in iOS framework tar", lname)
}
