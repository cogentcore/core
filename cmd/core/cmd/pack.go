// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	_ "embed"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"cogentcore.org/core/base/exec"
	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/base/strcase"
	"cogentcore.org/core/cmd/core/config"
	"cogentcore.org/core/cmd/core/rendericon"
	"github.com/jackmordaunt/icns/v2"
)

// Pack builds and packages the app for the target platform.
// For android, ios, and web, it is equivalent to build.
func Pack(c *config.Config) error { //types:add
	err := Build(c)
	if err != nil {
		return err
	}
	for _, platform := range c.Build.Target {
		switch platform.OS {
		case "android", "ios", "web": // build already packages
			continue
		case "linux":
			err := packLinux(c)
			if err != nil {
				return err
			}
		case "darwin":
			err := packDarwin(c)
			if err != nil {
				return err
			}
		case "windows":
			err := packWindows(c)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// packLinux packages the app for Linux by generating a .deb file.
func packLinux(c *config.Config) error {
	// based on https://ubuntuforums.org/showthread.php?t=910717

	anm := strcase.ToKebab(c.Name)
	vnm := strings.TrimPrefix(c.Version, "v")
	avnm := anm + "_" + vnm

	bpath := c.Build.Output
	apath := filepath.Join(bpath, avnm)
	ulbpath := filepath.Join(apath, "usr", "local", "bin")
	usipath := filepath.Join(apath, "usr", "share", "icons", "hicolor")
	usapath := filepath.Join(apath, "usr", "share", "applications")
	dpath := filepath.Join(apath, "DEBIAN")

	err := os.MkdirAll(ulbpath, 0777)
	if err != nil {
		return err
	}
	err = exec.Run("cp", "-p", c.Name, filepath.Join(ulbpath, anm))
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
	err = imagex.Save(ic, filepath.Join(i48path, anm+".png"))
	if err != nil {
		return err
	}
	iscpath := filepath.Join(usipath, "scalable", "apps")
	err = os.MkdirAll(iscpath, 0777)
	if err != nil {
		return err
	}
	err = exec.Run("cp", "icon.svg", filepath.Join(iscpath, anm+".svg"))
	if err != nil {
		return err
	}

	// we need a description
	if c.About == "" {
		c.About = c.Name
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
	dfd := &desktopFileData{
		Name: c.Name,
		Desc: c.About,
		Exec: anm,
	}
	err = desktopFileTmpl.Execute(fapp, dfd)
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
	dcd := &debianControlData{
		Name:    anm,
		Version: vnm,
		Desc:    c.About,
	}
	err = debianControlTmpl.Execute(fctrl, dcd)
	if err != nil {
		return err
	}
	return exec.Run("dpkg-deb", "--build", apath)
}

// desktopFileData is the data passed to [desktopFileTmpl]
type desktopFileData struct {
	Name string
	Desc string
	Exec string
}

// TODO(kai): project website

// desktopFileTmpl is the template for the Linux .desktop file
var desktopFileTmpl = template.Must(template.New("desktopFileTmpl").Parse(
	`[Desktop Entry]
Type=Application
Version=1.0
Name={{.Name}}
Comment={{.Desc}}
Exec={{.Exec}}
Icon={{.Exec}}
Terminal=false
`))

// debianControlData is the data passed to [debianControlTmpl]
type debianControlData struct {
	Name    string
	Version string
	Desc    string
}

// TODO(kai): architecture, maintainer, dependencies

// debianControlTmpl is the template for the Linux DEBIAN/control file
var debianControlTmpl = template.Must(template.New("debianControlTmpl").Parse(
	`Package: {{.Name}}
Version: {{.Version}}
Section: base
Priority: optional
Architecture: all
Maintainer: Your Name <you@email.com>
Description: {{.Desc}}
`))

// packDarwin packages the app for macOS by generating a .app and .dmg file.
func packDarwin(c *config.Config) error {
	// based on https://github.com/machinebox/appify

	anm := c.Name + ".app"

	bpath := c.Build.Output
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

	err = exec.Run("cp", "-p", c.Name, filepath.Join(mpath, anm))
	if err != nil {
		return err
	}
	err = exec.Run("chmod", "+x", mpath)
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
	ipd := &infoPlistData{
		Name:               c.Name,
		Executable:         filepath.Join("MacOS", anm),
		Identifier:         c.ID,
		Version:            c.Version,
		InfoString:         c.About,
		ShortVersionString: c.Version,
		IconFile:           filepath.Join("Contents", "Resources", "icon.icns"),
	}
	err = infoPlistTmpl.Execute(fplist, ipd)
	if err != nil {
		return err
	}

	if !c.Pack.DMG {
		return nil
	}
	// install dmgbuild if we don't already have it
	if _, err := exec.LookPath("dmgbuild"); err != nil {
		err = exec.Verbose().SetBuffer(false).Run("pip", "install", "dmgbuild")
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
	dmgbd := &dmgBuildData{
		AppPath:  apath,
		AppName:  anm,
		IconPath: inm,
	}
	err = dmgBuildTmpl.Execute(fdmg, dmgbd)
	if err != nil {
		return err
	}
	err = exec.Run("dmgbuild",
		"-s", dmgsnm,
		c.Name, filepath.Join(bpath, c.Name+".dmg"))
	if err != nil {
		return err
	}
	return os.Remove(dmgsnm)
}

// infoPlistData is the data passed to [infoPlistTmpl]
type infoPlistData struct {
	Name               string
	Executable         string
	Identifier         string
	Version            string
	InfoString         string
	ShortVersionString string
	IconFile           string
}

// infoPlistTmpl is the template for the macOS .app Info.plist
var infoPlistTmpl = template.Must(template.New("infoPlistTmpl").Parse(
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

// dmgBuildData is the data passed to [dmgBuildTmpl]
type dmgBuildData struct {
	AppPath  string
	AppName  string
	IconPath string
}

// dmgBuildTmpl is the template for the dmgbuild python settings file
var dmgBuildTmpl = template.Must(template.New("dmgBuildTmpl").Parse(
	`files = ['{{.AppPath}}']
symlinks = {"Applications": "/Applications"}
icon = '{{.IconPath}}'
icon_locations = {'{{.AppName}}': (140, 120), "Applications": (500, 120)}
background = "builtin-arrow"
`))

// packWindows packages the app for Windows by generating a .msi file.
func packWindows(c *config.Config) error {
	opath := c.Build.Output
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
	wmd := &windowsInstallerData{
		Name: c.Name,
		Desc: c.About,
	}
	err = windowsInstallerTmpl.Execute(fman, wmd)
	fman.Close()
	if err != nil {
		return err
	}

	err = exec.Run("cp", c.Name+".exe", filepath.Join(ipath, "app.exe"))
	if err != nil {
		return err
	}
	err = exec.Run("cp", "icon.svg", filepath.Join(ipath, "icon.svg"))
	if err != nil {
		return err
	}

	err = exec.Run("go", "build", "-o", epath, gpath)
	if err != nil {
		return err
	}

	return os.RemoveAll(ipath)
}

// windowsInstallerData is the data passed to [windowsInstallerTmpl]
type windowsInstallerData struct {
	Name string
	Desc string
}

//go:embed windowsinstaller.go.tmpl
var windowsInstallerTmplString string

// windowsInstallerTmpl is the template for the Windows installer Go file
var windowsInstallerTmpl = template.Must(template.New("windowsInstallerTmpl").Parse(windowsInstallerTmplString))
