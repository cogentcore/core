// Copyright (c) 2023, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tools

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/iancoleman/strcase"
	"github.com/jackmordaunt/icns/v2"
	"goki.dev/goki/config"
	"goki.dev/goki/mobile"
	"goki.dev/xe"
)

// Pack builds and packages the app for the target platform.
// For android, ios, and js, it is equivalent to build.
func Pack(c *config.Config) error { //gti:add
	err := Build(c)
	if err != nil {
		return err
	}
	for _, platform := range c.Build.Target {
		switch platform.OS {
		case "android", "ios", "js": // build already packages
			continue
		case "linux":
			err := PackLinux(c)
			if err != nil {
				return err
			}
		case "darwin":
			err := PackDarwin(c)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// PackLinux packages the app for Linux by generating a .deb file.
func PackLinux(c *config.Config) error {
	// based on https://ubuntuforums.org/showthread.php?t=910717

	anm := strings.ToLower(strcase.ToCamel(c.Name))
	vnm := strings.TrimPrefix(c.Version, "v")
	avnm := anm + "_" + vnm

	bpath := filepath.Join(".goki", "bin", "linux")
	apath := filepath.Join(bpath, avnm)
	ubpath := filepath.Join(apath, "usr", "local", "bin")
	dpath := filepath.Join(apath, "DEBIAN")

	err := os.MkdirAll(ubpath, 0777)
	if err != nil {
		return err
	}
	err = xe.Run("cp", "-p", c.Build.Output, filepath.Join(ubpath, anm))
	if err != nil {
		return err
	}

	err = os.MkdirAll(dpath, 0777)
	if err != nil {
		return err
	}
	fctrl, err := os.Create(filepath.Join(dpath, "control"))
	if err != nil {
		return err
	}
	defer fctrl.Close()
	dcd := &DebianControlData{
		Name:        anm,
		Version:     vnm,
		Description: c.Desc,
	}
	// we need a description
	if dcd.Description == "" {
		dcd.Description = c.Name
	}
	err = DebianControlTmpl.Execute(fctrl, dcd)
	if err != nil {
		return err
	}
	return xe.Run("dpkg-deb", "--build", apath)
}

// DebianControlData is the data passed to [DebianControlTmpl]
type DebianControlData struct {
	Name        string
	Version     string
	Description string
}

// TODO(kai): architecture, maintainer, dependencies, description

// DebianControlTmpl is the template for the Linux DEBIAN/control file
var DebianControlTmpl = template.Must(template.New("DebianControlTmpl").Parse(
	`Package: {{.Name}}
Version: {{.Version}}
Section: base
Priority: optional
Architecture: all
Maintainer: Your Name <you@email.com>
Description: {{.Description}}
`))

// PackDarwin packages the app for macOS by generating a .app and .dmg file.
func PackDarwin(c *config.Config) error {
	// based on https://github.com/machinebox/appify

	anm := c.Name + ".app"

	bpath := filepath.Join(".goki", "bin", "darwin")
	apath := filepath.Join(bpath, anm)
	cpath := filepath.Join(apath, "Contents")
	mpath := filepath.Join(cpath, "MacOS")
	rpath := filepath.Join(cpath, "Resources")

	err := os.MkdirAll(mpath, 0777)
	if err != nil {
		return err
	}
	err = os.MkdirAll(rpath, 0777)
	if err != nil {
		return err
	}

	err = xe.Run("cp", "-p", c.Build.Output, filepath.Join(mpath, anm))
	if err != nil {
		return err
	}
	err = xe.Run("chmod", "+x", mpath)
	if err != nil {
		return err
	}

	inm := filepath.Join(rpath, "icon.icns")
	fdsi, err := os.Create(inm)
	if err != nil {
		return err
	}
	defer fdsi.Close()
	// 1024x1024 is the largest icon size on macOS
	sic, err := mobile.RenderIcon(1024)
	if err != nil {
		return err
	}
	err = icns.Encode(fdsi, sic)
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
		Executable:         filepath.Join("MacOS", anm),
		Identifier:         c.ID,
		Version:            c.Version,
		InfoString:         c.Desc,
		ShortVersionString: c.Version,
		IconFile:           filepath.Join("Contents", "Resources", "icon.icns"),
	}
	err = InfoPlistTmpl.Execute(fplist, ipd)
	if err != nil {
		return err
	}

	if !c.Pack.DMG {
		return nil
	}
	// install dmgbuild if we don't already have it
	if _, err := exec.LookPath("dmgbuild"); err != nil {
		err = xe.Run("pip", "install", "dmgbuild")
		if err != nil {
			return err
		}
	}
	dmgsnm := filepath.Join(bpath, ".tempDmgBuildSettings.py")
	fdmg, err := os.Create(dmgsnm)
	if err != nil {
		return err
	}
	defer fdmg.Close()
	dmgbd := &DmgBuildData{
		AppPath:  apath,
		AppName:  anm,
		IconPath: inm,
	}
	err = DmgBuildTmpl.Execute(fdmg, dmgbd)
	if err != nil {
		return err
	}
	err = xe.Run("dmgbuild",
		"-s", dmgsnm,
		c.Name, filepath.Join(bpath, c.Name+".dmg"))
	if err != nil {
		return err
	}
	return os.Remove(dmgsnm)
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

// DmgBuildData is the data passed to [DmgBuildTmpl]
type DmgBuildData struct {
	AppPath  string
	AppName  string
	IconPath string
}

// DmgBuildTmpl is the template for the dmgbuild python settings file
var DmgBuildTmpl = template.Must(template.New("DmgBuildTmpl").Parse(
	`files = ['{{.AppPath}}']
symlinks = {"Applications": "/Applications"}
icon = '{{.IconPath}}'
icon_locations = {'{{.AppName}}': (140, 120), "Applications": (500, 120)}
background = "builtin-arrow"
`))
