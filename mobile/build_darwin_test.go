// Copyright 2015 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"text/template"

	"goki.dev/goki/config"
	"goki.dev/grease"
)

func TestAppleBuild(t *testing.T) {
	if !xcodeAvailable() {
		t.Skip("Xcode is missing")
	}
	c := &config.Config{}
	grease.SetFromDefaults(c)
	defer func() {
		Xout = os.Stderr
		c.Build.PrintOnly = false
		c.Build.Print = false
	}()
	c.Build.PrintOnly = true
	c.Build.Print = true
	c.Build.Target = []config.Platform{config.Platform{OS: "ios", Arch: "arm64"}}
	c.Build.BundleID = "org.golang.todo"
	gopath = filepath.SplitList(goEnv("GOPATH"))[0]
	oldTags := c.Build.Tags
	c.Build.Tags = []string{"tag1"}
	defer func() {
		c.Build.Tags = oldTags
	}()
	tests := []struct {
		pkg  string
		main bool
	}{
		{"goki.dev/mobile/example/basic", true},
		{"goki.dev/mobile/bind/testdata/testpkg", false},
	}
	for _, test := range tests {
		buf := new(bytes.Buffer)
		Xout = buf
		var tmpl *template.Template
		if test.main {
			c.Build.Output = "basic.app"
			tmpl = appleMainBuildTmpl
		} else {
			c.Build.Output = ""
			tmpl = appleOtherBuildTmpl
		}
		c.Build.Package = test.pkg
		err := Build(c)
		if err != nil {
			t.Log(buf.String())
			t.Fatal(err)
		}

		teamID, err := DetectTeamID()
		if err != nil {
			t.Fatalf("detecting team ID failed: %v", err)
		}

		output, err := defaultOutputData(teamID)
		if err != nil {
			t.Fatal(err)
		}

		data := struct {
			outputData
			TeamID string
			Pkg    string
			Main   bool
			BuildO string
		}{
			outputData: output,
			TeamID:     teamID,
			Pkg:        test.pkg,
			Main:       test.main,
			BuildO:     c.Build.Output,
		}

		got := filepath.ToSlash(buf.String())

		wantBuf := new(bytes.Buffer)

		if err := tmpl.Execute(wantBuf, data); err != nil {
			t.Fatalf("computing diff failed: %v", err)
		}

		diff, err := diff(got, wantBuf.String())

		if err != nil {
			t.Fatalf("computing diff failed: %v", err)
		}
		if diff != "" {
			t.Errorf("unexpected output:\n%s", diff)
		}
	}
}

var appleMainBuildTmpl = template.Must(InfoplistTmpl.New("output").Parse(`GOMOBILE={{.GOPATH}}/pkg/gomobile
WORK=$WORK
mkdir -p $WORK/main.xcodeproj
echo "{{.Xproj}}" > $WORK/main.xcodeproj/project.pbxproj
mkdir -p $WORK/main
echo "{{template "infoplist" .Xinfo}}" > $WORK/main/Info.plist
mkdir -p $WORK/main/Images.xcassets/AppIcon.appiconset
echo "{{.Xcontents}}" > $WORK/main/Images.xcassets/AppIcon.appiconset/Contents.json
GOMODCACHE=$GOPATH/pkg/mod GOOS=ios GOARCH=arm64 GOFLAGS=-tags=ios CC=iphoneos-clang CXX=iphoneos-clang++ CGO_CFLAGS=-isysroot iphoneos -miphoneos-version-min=13.0  -arch arm64 CGO_CXXFLAGS=-isysroot iphoneos -miphoneos-version-min=13.0  -arch arm64 CGO_LDFLAGS=-isysroot iphoneos -miphoneos-version-min=13.0  -arch arm64 CGO_ENABLED=1 DARWIN_SDK=iphoneos go build -tags tag1 -x -ldflags=-w -o=$WORK/ios/arm64 {{.Pkg}}
GOMODCACHE=$GOPATH/pkg/mod GOOS=ios GOARCH=amd64 GOFLAGS=-tags=ios CC=iphonesimulator-clang CXX=iphonesimulator-clang++ CGO_CFLAGS=-isysroot iphonesimulator -mios-simulator-version-min=13.0  -arch x86_64 CGO_CXXFLAGS=-isysroot iphonesimulator -mios-simulator-version-min=13.0  -arch x86_64 CGO_LDFLAGS=-isysroot iphonesimulator -mios-simulator-version-min=13.0  -arch x86_64 CGO_ENABLED=1 DARWIN_SDK=iphonesimulator go build -tags tag1 -x -ldflags=-w -o=$WORK/iossimulator/amd64 {{.Pkg}}
xcrun lipo -o $WORK/main/main -create $WORK/ios/arm64 $WORK/iossimulator/amd64
mkdir -p $WORK/main/assets
xcrun xcodebuild -configuration Release -project $WORK/main.xcodeproj -allowProvisioningUpdates DEVELOPMENT_TEAM={{.TeamID}}
mv $WORK/build/Release-iphoneos/main.app {{.BuildO}}
`))

var appleOtherBuildTmpl = template.Must(InfoplistTmpl.New("output").Parse(`GOMOBILE={{.GOPATH}}/pkg/gomobile
WORK=$WORK
GOMODCACHE=$GOPATH/pkg/mod GOOS=ios GOARCH=arm64 GOFLAGS=-tags=ios CC=iphoneos-clang CXX=iphoneos-clang++ CGO_CFLAGS=-isysroot iphoneos -miphoneos-version-min=13.0  -arch arm64 CGO_CXXFLAGS=-isysroot iphoneos -miphoneos-version-min=13.0  -arch arm64 CGO_LDFLAGS=-isysroot iphoneos -miphoneos-version-min=13.0  -arch arm64 CGO_ENABLED=1 DARWIN_SDK=iphoneos go build -tags tag1 -x {{.Pkg}}
GOMODCACHE=$GOPATH/pkg/mod GOOS=ios GOARCH=arm64 GOFLAGS=-tags=ios CC=iphonesimulator-clang CXX=iphonesimulator-clang++ CGO_CFLAGS=-isysroot iphonesimulator -mios-simulator-version-min=13.0  -arch arm64 CGO_CXXFLAGS=-isysroot iphonesimulator -mios-simulator-version-min=13.0  -arch arm64 CGO_LDFLAGS=-isysroot iphonesimulator -mios-simulator-version-min=13.0  -arch arm64 CGO_ENABLED=1 DARWIN_SDK=iphonesimulator go build -tags tag1 -x {{.Pkg}}
GOMODCACHE=$GOPATH/pkg/mod GOOS=ios GOARCH=amd64 GOFLAGS=-tags=ios CC=iphonesimulator-clang CXX=iphonesimulator-clang++ CGO_CFLAGS=-isysroot iphonesimulator -mios-simulator-version-min=13.0  -arch x86_64 CGO_CXXFLAGS=-isysroot iphonesimulator -mios-simulator-version-min=13.0  -arch x86_64 CGO_LDFLAGS=-isysroot iphonesimulator -mios-simulator-version-min=13.0  -arch x86_64 CGO_ENABLED=1 DARWIN_SDK=iphonesimulator go build -tags tag1 -x {{.Pkg}}
`))
