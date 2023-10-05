// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tools

import (
	"archive/tar"
	"fmt"
	"io"
	"net/http"

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
	resp, err := http.Get("https://github.com/KhronosGroup/MoltenVK/releases/download/latest/MoltenVK-ios.tar")
	if err != nil {
		return fmt.Errorf("error downloading iOS framework tar from latest MoltenVK release: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("got status code %d when downloading iOS framework tar from latest MoltenVK release", resp.StatusCode)
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
		fmt.Println(hdr.Name)
	}
	return nil
}
