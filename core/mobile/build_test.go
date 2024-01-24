// Copyright 2015 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// TODO(kai): maybe fix these tests at some point

package mobile

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"text/template"

	"cogentcore.org/core/core/config"
	"cogentcore.org/core/core/mobile/sdkpath"
	"cogentcore.org/core/grr"
	"cogentcore.org/core/laser"
	"cogentcore.org/core/xe"
)

func TestRFC1034Label(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"a", "a"},
		{"123", "-23"},
		{"a.b.c", "a-b-c"},
		{"a-b", "a-b"},
		{"a:b", "a-b"},
		{"a?b", "a-b"},
		{"αβγ", "---"},
		{"💩", "--"},
		{"My App", "My-App"},
		{"...", ""},
		{".-.", "--"},
	}

	for _, tc := range tests {
		if got := RFC1034Label(tc.in); got != tc.want {
			t.Errorf("rfc1034Label(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestAndroidPkgName(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"a", "a"},
		{"a123", "a123"},
		{"a.b.c", "a_b_c"},
		{"a-b", "a_b"},
		{"a:b", "a_b"},
		{"a?b", "a_b"},
		{"αβγ", "go___"},
		{"💩", "go_"},
		{"My App", "My_App"},
		{"...", "go___"},
		{".-.", "go___"},
		{"abstract", "abstract_"},
		{"Abstract", "Abstract"},
		{"12345", "go12345"},
	}

	for _, tc := range tests {
		if got := AndroidPkgName(tc.in); got != tc.want {
			t.Errorf("len %d", len(tc.in))
			t.Errorf("androidPkgName(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestAndroidBuild(t *testing.T) {
	t.Skip("not working and not worth it")
	if runtime.GOOS == "android" || runtime.GOOS == "ios" {
		t.Skipf("not available on %s", runtime.GOOS)
	}
	pout := os.Stdout
	r, w, err := os.Pipe()
	grr.TestFatal(t, err)
	os.Stdout = w

	c := &config.Config{}
	laser.SetFromDefaultTags(c)
	defer func() {
		xe.SetMajor(nil)
	}()
	c.Build.Target = []config.Platform{{OS: "android", Arch: "arm"}}
	gopath = filepath.ToSlash(filepath.SplitList(GoEnv("GOPATH"))[0])
	if GOOS == "windows" {
		os.Setenv("HOMEDRIVE", "C:")
	}
	c.ID = "org.golang.todo"
	pdir, err := os.Getwd()
	grr.Test(t, err)
	grr.Test(t, os.Chdir(filepath.Join("..", "..", "goosi", "examples", "drawtri")))
	err = Build(c)
	if err != nil {
		t.Fatal(err)
	}
	grr.Test(t, os.Chdir(pdir))

	grr.Test(t, w.Close())
	os.Stdout = pout
	buf := &bytes.Buffer{}
	_, err = io.Copy(buf, r)
	grr.TestFatal(t, err)

	diff, err := diffOutput(buf.String(), androidBuildTmpl)
	if err != nil {
		t.Fatalf("computing diff failed: %v", err)
	}
	if diff != "" {
		t.Errorf("unexpected output:\n%s", diff)
	}
}

var androidBuildTmpl = template.Must(template.New("output").Parse(`GOMOBILE={{.GOPATH}}/pkg/gomobile
WORK=$WORK
mkdir -p $WORK/lib/armeabi-v7a
GOMODCACHE=$GOPATH/pkg/mod GOOS=android GOARCH=arm CC=$NDK_PATH/toolchains/llvm/prebuilt/{{.NDKARCH}}/bin/armv7a-linux-androideabi16-clang CXX=$NDK_PATH/toolchains/llvm/prebuilt/{{.NDKARCH}}/bin/armv7a-linux-androideabi16-clang++ CGO_ENABLED=1 GOARM=7 go build -tags tag1 -x -buildmode=c-shared -o $WORK/lib/armeabi-v7a/libbasic.so golang.org/x/mobile/example/basic
`))

// TODO: move this to config/platform

// func TestParseBuildTarget(t *testing.T) {
// 	wantAndroid := "android/" + strings.Join(platformArchs("android"), ",android/")

// 	tests := []struct {
// 		in      string
// 		wantErr bool
// 		want    string
// 	}{
// 		{"android", false, wantAndroid},
// 		{"android,android/arm", false, wantAndroid},
// 		{"android/arm", false, "android/arm"},

// 		{"ios", false, "ios/arm64,iossimulator/arm64,iossimulator/amd64"},
// 		{"ios,ios/arm64", false, "ios/arm64"},
// 		{"ios/arm64", false, "ios/arm64"},

// 		{"iossimulator", false, "iossimulator/arm64,iossimulator/amd64"},
// 		{"iossimulator/amd64", false, "iossimulator/amd64"},

// 		{"macos", false, "macos/arm64,macos/amd64"},
// 		{"macos,ios/arm64", false, "macos/arm64,macos/amd64,ios/arm64"},
// 		{"macos/arm64", false, "macos/arm64"},
// 		{"macos/amd64", false, "macos/amd64"},

// 		{"maccatalyst", false, "maccatalyst/arm64,maccatalyst/amd64"},
// 		{"maccatalyst,ios/arm64", false, "maccatalyst/arm64,maccatalyst/amd64,ios/arm64"},
// 		{"maccatalyst/arm64", false, "maccatalyst/arm64"},
// 		{"maccatalyst/amd64", false, "maccatalyst/amd64"},

// 		{"", true, ""},
// 		{"linux", true, ""},
// 		{"android/x86", true, ""},
// 		{"android/arm5", true, ""},
// 		{"ios/mips", true, ""},
// 		{"android,ios", true, ""},
// 		{"ios,android", true, ""},
// 		{"ios/amd64", true, ""},
// 	}

// 	for _, tc := range tests {
// 		t.Run(tc.in, func(t *testing.T) {
// 			targets := []config.Platform{}
// 			for _, tin := range strings.Split(tc.in, ",") {
// 				tar, err := config.ParsePlatform(tin)
// 				if err != nil {
// 					t.Error(err)
// 				}
// 				targets = append(targets, tar)
// 			}
// 			var s []string
// 			for _, t := range targets {
// 				s = append(s, t.String())
// 			}
// 			got := strings.Join(s, ",")
// 			if tc.wantErr {
// 				if err == nil {
// 					t.Errorf("-target=%q; want error, got (%q, nil)", tc.in, got)
// 				}
// 				return
// 			}
// 			if err != nil || got != tc.want {
// 				t.Errorf("-target=%q; want (%q, nil), got (%q, %v)", tc.in, tc.want, got, err)
// 			}
// 		})
// 	}
// }

func TestRegexImportGolangXPackage(t *testing.T) {
	tests := []struct {
		in      string
		want    string
		wantLen int
	}{
		{"ffffffff t golang.org/x/mobile", "golang.org/x/mobile", 2},
		{"ffffffff t github.com/example/repo/vendor/golang.org/x/mobile", "golang.org/x/mobile", 2},
		{"ffffffff t github.com/example/golang.org/x/mobile", "", 0},
		{"ffffffff t github.com/example/repo", "", 0},
		{"ffffffff t github.com/example/repo/vendor", "", 0},
		{"ffffffff t _golang.org/x/mobile/app", "golang.org/x/mobile/app", 2},
	}

	for _, tc := range tests {
		res := NmRE.FindStringSubmatch(tc.in)
		if len(res) != tc.wantLen {
			t.Errorf("nmRE returned unexpected result for %q: want len(res) = %d, got %d",
				tc.in, tc.wantLen, len(res))
			continue
		}
		if tc.wantLen == 0 {
			continue
		}
		if res[1] != tc.want {
			t.Errorf("nmRE returned unexpected result. want (%v), got (%v)",
				tc.want, res[1])
		}
	}
}

func TestBuildWithGoModules(t *testing.T) {
	if runtime.GOOS == "android" || runtime.GOOS == "ios" {
		t.Skipf("gomobile are not available on %s", runtime.GOOS)
	}
	dir, err := os.MkdirTemp("", "gomobile-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	if out, err := exec.Command("go", "build", "-o="+dir, "cogentcore.org/core/goosi/examples/drawtri").CombinedOutput(); err != nil {
		t.Fatalf("%v: %s", err, string(out))
	}
	path := dir
	if p := os.Getenv("PATH"); p != "" {
		path += string(filepath.ListSeparator) + p
	}

	for _, target := range []string{"android", "ios"} {
		t.Run(target, func(t *testing.T) {
			switch target {
			case "android":
				if _, err := sdkpath.AndroidAPIPath(MinAndroidSDK); err != nil {
					t.Skip("No compatible android API platform found, skipping bind")
				}
			case "ios":
				if !XCodeAvailable() {
					t.Skip("Xcode is missing")
				}
			}

			tests := []struct {
				Name string
				Path string
			}{
				{
					Name: "Relative Path",
					Path: filepath.Join("..", "..", "goosi", "examples", "drawtri"),
				},
			}

			for _, tc := range tests {
				tc := tc
				t.Run(tc.Name, func(t *testing.T) {
					c := &config.Config{}
					laser.SetFromDefaultTags(c)
					c.ID = "org.golang.todo"
					c.Build.Target = make([]config.Platform, 1)
					grr.Test(t, c.Build.Target[0].SetString(target))
					pdir, err := os.Getwd()
					grr.Test(t, err)
					grr.Test(t, os.Chdir(tc.Path))
					grr.Test(t, Build(c))
					grr.Test(t, os.Chdir(pdir))
				})
			}
		})
	}
}
