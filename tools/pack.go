// Copyright (c) 2023, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tools

import (
	"os"
	"path/filepath"
	"text/template"

	"goki.dev/goki/config"
	"goki.dev/goki/packman"
	"goki.dev/xe"
)

// Pack builds and packages the app for the target platform.
// For android, ios, and js, it is equivalent to build.
func Pack(c *config.Config) error { //gti:add
	err := packman.Build(c)
	if err != nil {
		return err
	}
	err = os.MkdirAll(filepath.Join(".goki", "bin", "pack"), 0777)
	if err != nil {
		return err
	}
	for _, platform := range c.Build.Target {
		switch platform.OS {
		case "android", "ios", "js": // build already packages
			continue
		case "darwin":
			err := PackDarwin(c)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// PackDarwin packages the app for macOS.
func PackDarwin(c *config.Config) error {
	apath := filepath.Join(".goki", "bin", "pack", c.Name+".app")
	cpath := filepath.Join(apath, "Contents")
	mpath := filepath.Join(cpath, "MacOS")
	rpath := filepath.Join(cpath, "Resources")

	err := os.MkdirAll(mpath, 0777)
	if err != nil {
		return err
	}

	err = xe.Run("cp", filepath.Join(".goki", "bin", "build", c.Name), mpath)
	if err != nil {
		return err
	}
	err = xe.Run("chmod", "+x", mpath)
	if err != nil {
		return err
	}
	err = xe.Run("chmod", "+x", filepath.Join(mpath, c.Name))
	if err != nil {
		return err
	}

	fplist, err := os.Create(filepath.Join(cpath, "Info.plist"))
	if err != nil {
		return err
	}
	defer fplist.Close()
	ipd := &InfoPlistData{
		Name:               c.Name,
		Executable:         filepath.Join("MacOS", c.Name+".app"),
		Identifier:         c.Build.ID,
		Version:            c.Version,
		InfoString:         c.Desc,
		ShortVersionString: c.Version,
		IconFile:           filepath.Join(rpath, "icon.icns"),
	}
	err = InfoPlistTmpl.Execute(fplist, ipd)
	if err != nil {
		return err
	}
	return nil
}

// InfoPlistData is the data passed to [InfoPlistTmpl]
type InfoPlistData struct {
	Name               string
	Executable         string
	Identifier         string
	Version            string
	InfoString         string
	ShortVersionString string
	IconFile           string
}

// InfoPlistTmpl is the template for the macOS .app Info.plist
var InfoPlistTmpl = template.Must(template.New("InfoPlistTmpl").Parse(
	`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
	<dict>
		<key>CFBundlePackageType</key>
		<string>APPL</string>
		<key>CFBundleInfoDictionaryVersion</key>
		<string>6.0</string>
		<key>CFBundleName</key>
		<string>{{ .Name }}</string>
		<key>CFBundleExecutable</key>
		<string>{{ .Executable }}</string>
		<key>CFBundleIdentifier</key>
		<string>{{ .Identifier }}</string>
		<key>CFBundleVersion</key>
		<string>{{ .Version }}</string>
		<key>CFBundleGetInfoString</key>
		<string>{{ .InfoString }}</string>
		<key>CFBundleShortVersionString</key>
		<string>{{ .ShortVersionString }}</string>
		<key>CFBundleIconFile</key>
		<string>{{ .IconFile }}</string>
	</dict>
</plist>
`))
