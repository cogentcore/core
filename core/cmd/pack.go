// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	_ "embed"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"cogentcore.org/core/core/config"
	"cogentcore.org/core/core/rendericon"
	"cogentcore.org/core/grows/images"
	"cogentcore.org/core/strcase"
	"cogentcore.org/core/xe"
	"github.com/jackmordaunt/icns/v2"
)

// Pack builds and packages the app for the target platform.
// For android, ios, and web, it is equivalent to build.
func Pack(c *config.Config) error { //gti:add
	err := Build(c)
	if err != nil {
		return err
	}
	for _, platform := range c.Build.Target {
		switch platform.OS {
		case "android", "ios", "web": // build already packages
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
		case "windows":
			err := PackWindows(c)
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

	anm := strcase.ToKebab(c.Name)
	vnm := strings.TrimPrefix(c.Version, "v")
	avnm := anm + "_" + vnm

	bpath := filepath.Join(".core", "bin", "linux")
	apath := filepath.Join(bpath, avnm)
	ulbpath := filepath.Join(apath, "usr", "local", "bin")
	usipath := filepath.Join(apath, "usr", "share", "icons", "hicolor")
	usapath := filepath.Join(apath, "usr", "share", "applications")
	dpath := filepath.Join(apath, "DEBIAN")

	err := os.MkdirAll(ulbpath, 0777)
	if err != nil {
		return err
	}
	err = xe.Run("cp", "-p", filepath.Join(bpath, c.Name), filepath.Join(ulbpath, anm))
	if err != nil {
		return err
	}

	// see https://martin.hoppenheit.info/blog/2016/where-to-put-application-icons-on-linux/
	// TODO(kai): consider rendering more icon sizes and/or an XPM icon
	ic, err := rendericon.Render(48)
	if err != nil {
		return err
	}
	i48path := filepath.Join(usipath, "48x48", "apps")
	err = os.MkdirAll(i48path, 0777)
	if err != nil {
		return err
	}
	err = images.Save(ic, filepath.Join(i48path, anm+".png"))
	if err != nil {
		return err
	}
	iscpath := filepath.Join(usipath, "scalable", "apps")
	err = os.MkdirAll(iscpath, 0777)
	if err != nil {
		return err
	}
	err = xe.Run("cp", filepath.Join(".core", "icon.svg"), filepath.Join(iscpath, anm+".svg"))
	if err != nil {
		return err
	}

	// we need a description
	if c.Desc == "" {
		c.Desc = c.Name
	}

	err = os.MkdirAll(usapath, 0777)
	if err != nil {
		return err
	}
	fapp, err := os.Create(filepath.Join(usapath, anm+".desktop"))
	if err != nil {
		return err
	}
	defer fapp.Close()
	dfd := &DesktopFileData{
		Name: c.Name,
		Desc: c.Desc,
		Exec: anm,
	}
	err = DesktopFileTmpl.Execute(fapp, dfd)
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
		Name:    anm,
		Version: vnm,
		Desc:    c.Desc,
	}
	err = DebianControlTmpl.Execute(fctrl, dcd)
	if err != nil {
		return err
	}
	return xe.Run("dpkg-deb", "--build", apath)
}

// DesktopFileData is the data passed to [DesktopFileTmpl]
type DesktopFileData struct {
	Name string
	Desc string
	Exec string
}

// TODO(kai): project website

// DesktopFileTmpl is the template for the Linux .desktop file
var DesktopFileTmpl = template.Must(template.New("DesktopFileTmpl").Parse(
	`[Desktop Entry]
Type=Application
Version=1.0
Name={{.Name}}
Comment={{.Desc}}
Exec={{.Exec}}
Icon={{.Exec}}
Terminal=false
`))

// DebianControlData is the data passed to [DebianControlTmpl]
type DebianControlData struct {
	Name    string
	Version string
	Desc    string
}

// TODO(kai): architecture, maintainer, dependencies

// DebianControlTmpl is the template for the Linux DEBIAN/control file
var DebianControlTmpl = template.Must(template.New("DebianControlTmpl").Parse(
	`Package: {{.Name}}
Version: {{.Version}}
Section: base
Priority: optional
Architecture: all
Maintainer: Your Name <you@email.com>
Description: {{.Desc}}
`))

// PackDarwin packages the app for macOS by generating a .app and .dmg file.
func PackDarwin(c *config.Config) error {
	// based on https://github.com/machinebox/appify

	anm := c.Name + ".app"

	bpath := filepath.Join(".core", "bin", "darwin")
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

	err = xe.Run("cp", "-p", filepath.Join(bpath, c.Name), filepath.Join(mpath, anm))
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
	sic, err := rendericon.Render(1024)
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
		err = xe.Verbose().SetBuffer(false).Run("pip", "install", "dmgbuild")
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

// PackWindows packages the app for Windows by generating a .msi file.
func PackWindows(c *config.Config) error {
	opath := filepath.Join(".core", "bin", "windows")
	ipath := filepath.Join(opath, "tempWindowsInstaller")
	gpath := filepath.Join(ipath, "installer.go")
	epath := filepath.Join(opath, c.Name+" Installer.exe")

	err := os.MkdirAll(ipath, 0777)
	if err != nil {
		return err
	}

	fman, err := os.Create(gpath)
	if err != nil {
		return err
	}
	wmd := &WindowsInstallerData{
		Name: c.Name,
		Desc: c.Desc,
	}
	err = WindowsInstallerTmpl.Execute(fman, wmd)
	fman.Close()
	if err != nil {
		return err
	}

	err = xe.Run("cp", filepath.Join(".core", "bin", "windows", c.Name+".exe"), filepath.Join(ipath, "app.exe"))
	if err != nil {
		return err
	}
	err = xe.Run("cp", filepath.Join(".core", "icon.svg"), filepath.Join(ipath, "icon.svg"))
	if err != nil {
		return err
	}

	err = xe.Run("go", "build", "-o", epath, gpath)
	if err != nil {
		return err
	}

	return os.RemoveAll(ipath)
}

// WindowsInstallerData is the data passed to [WindowsInstallerTmpl]
type WindowsInstallerData struct {
	Name string
	Desc string
}

//go:embed windowsinstaller.go.tmpl
var windowsInstallerTmplString string

// WindowsInstallerTmpl is the template for the Windows installer Go file
var WindowsInstallerTmpl = template.Must(template.New("WindowsInstallerTmpl").Parse(windowsInstallerTmplString))
