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

	"goki.dev/goki/config"
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
	tlname := "/System/Library/Frameworks/__tmp_goki_setup_libMoltenVK.dylib"

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
		f, err := os.Create(tlname)
		if err != nil {
			return fmt.Errorf("error creating file for MoltenVK dylib: %w", err)
		}
		defer f.Close()
		_, err = io.Copy(f, tr)
		if err != nil {
			return fmt.Errorf("error copying MoltenVK dylib to MoltenVK dylib file: %w", err)
		}
		return nil
	}
	return fmt.Errorf("internal error: did not find MoltenVK dylib at %q in iOS framework tar", lname)
}
