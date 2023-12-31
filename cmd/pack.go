// Copyright (c) 2023, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

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
	"goki.dev/grows/images"
	"goki.dev/xe"

	// we need to depend on go-msi so that we can find its templates
	_ "github.com/mh-cbon/go-msi/util"
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

	bpath := filepath.Join(".goki", "bin", "linux")
	apath := filepath.Join(bpath, avnm)
	ulbpath := filepath.Join(apath, "usr", "local", "bin")
	usipath := filepath.Join(apath, "usr", "share", "icons", "hicolor")
	usapath := filepath.Join(apath, "usr", "share", "applications")
	dpath := filepath.Join(apath, "DEBIAN")

	err := os.MkdirAll(ulbpath, 0777)
	if err != nil {
		return err
	}
	err = xe.Run("cp", "-p", c.Build.Output, filepath.Join(ulbpath, anm))
	if err != nil {
		return err
	}

	// see https://martin.hoppenheit.info/blog/2016/where-to-put-application-icons-on-linux/
	// TODO(kai): consider rendering more icon sizes and/or an XPM icon
	ic, err := mobile.RenderIcon(48)
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
	err = xe.Run("cp", filepath.Join(".goki", "icon.svg"), filepath.Join(iscpath, anm+".svg"))
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
	// install go-msi if we don't already have it
	if _, err := exec.LookPath("go-msi"); err != nil {
		err := xe.Run("go", "install", "github.com/mh-cbon/go-msi@latest")
		if err != nil {
			return err
		}
	}
	// install wix if we don't already have it
	if _, err := exec.LookPath("wix"); err != nil {
		out, err := xe.Output("dotnet", "--list-sdks")
		if err != nil {
			return err
		}
		if len(strings.TrimSpace(out)) == 0 {
			err = xe.Verbose().SetBuffer(false).Run("winget", "install", "Microsoft.DotNet.SDK.8")
			if err != nil {
				return err
			}
		}
		err = xe.Verbose().SetBuffer(false).Run("dotnet", "tool", "install", "--global", "wix", "--version", "4.0.3")
		if err != nil {
			return err
		}
	}

	opath := filepath.Join(".goki", "bin", "windows")
	jpath := filepath.Join(opath, ".tempWixManifest.json")
	mpath := filepath.Join(opath, c.Name+".msi")

	fman, err := os.Create(jpath)
	if err != nil {
		return err
	}
	defer fman.Close()
	wmd := &WixManifestData{
		Name: c.Name,
		Exec: strings.ReplaceAll(c.Build.Output, `\`, `\\`), // need to escape
		Desc: c.Desc,
	}
	err = WixManifestTmpl.Execute(fman, wmd)
	if err != nil {
		return err
	}

	// see https://stackoverflow.com/questions/67211875/how-to-get-the-path-to-a-go-module-dependency
	goMsiPath, err := xe.Output("go", "list", "-m", "-f", "{{.Dir}}", "github.com/mh-cbon/go-msi")
	if err != nil {
		return err
	}

	err = xe.Run("go-msi", "make",
		"--path", jpath,
		"--msi", mpath,
		"--version", c.Version,
		"--src", filepath.Join(goMsiPath, "templates"))
	if err != nil {
		return err
	}

	return os.Remove(jpath)
}

// WixManifestData is the data passed to [WixManifestTmpl]
type WixManifestData struct {
	Name string
	Exec string
	Desc string
}

// WixManifestTmpl is the template for the go-msi wix manifest json file
var WixManifestTmpl = template.Must(template.New("WixManifestTmpl").Parse(
	`{
	"product": "{{.Name}}",
	"files": {
		"guid": "",
		"items": [
		"{{.Exec}}"
		]
	},
	"shortcuts": {
		"guid": "",
		"items": [
		{
			"name": "{{.Name}}",
			"description": "{{.Name}}",
			"target": "[INSTALLDIR]\\{{.Name}}.exe",
			"wdir": "INSTALLDIR",
			"icon":"ico.ico"
		}
		]
	},
	"hooks": [
		{"when": "install", "command": "[INSTALLDIR]\\{{.Name}}.exe"}
	],
	"choco": {
		"description": "{{.Desc}}"
	}
}
`))
