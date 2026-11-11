// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"cogentcore.org/core/base/exec"
	"cogentcore.org/core/base/logx"
)

// The video package (cogentcore.org/core/video) decodes video on desktop
// platforms with FFmpeg through go-astiav, which requires the development
// libraries of one major version of FFmpeg: [ffmpegVersion]. core setup
// -video installs them, and core update -video updates them:
//   - on macOS, with Homebrew (ffmpeg, or ffmpeg@9 once Homebrew has moved
//     on to a newer FFmpeg);
//   - on Linux and Windows (amd64 and arm64), as the prebuilt shared LGPL
//     builds of BtbN (https://github.com/BtbN/FFmpeg-Builds), which are
//     extracted into [ffmpegDir], as the FFmpeg packages of most systems
//     are older; FFmpeg of the system is used if it is the right version.
// The core command then builds and runs apps with the FFmpeg in
// [ffmpegDir] (see [SetFFmpegEnv]).

const (
	// ffmpegVersion is the version of FFmpeg that the go-astiav version of
	// the video package requires (see its README), which must be updated
	// with go-astiav in go.mod.
	ffmpegVersion = "9.0"

	// ffmpegAvutilMajor is the major version of libavutil of [ffmpegVersion],
	// by which an installed FFmpeg is checked.
	ffmpegAvutilMajor = 61

	// ffmpegReleases is the GitHub API URL of the release of BtbN that has
	// the latest builds of each version of FFmpeg, which are made daily.
	ffmpegReleases = "https://api.github.com/repos/BtbN/FFmpeg-Builds/releases/tags/latest"
)

// ffmpegMajor returns the major version of [ffmpegVersion].
func ffmpegMajor() string {
	major, _, _ := strings.Cut(ffmpegVersion, ".")
	return major
}

// ffmpegDir returns the directory that the FFmpeg of BtbN is installed in,
// in the cache directory of the user, or "" if there is none.
func ffmpegDir() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "cogentcore", "ffmpeg")
}

// ffmpegInstalled returns whether the FFmpeg of BtbN is installed in
// [ffmpegDir], with its pkg-config files.
func ffmpegInstalled() bool {
	dir := ffmpegDir()
	if dir == "" {
		return false
	}
	_, err := os.Stat(filepath.Join(dir, "lib", "pkgconfig", "libavutil.pc"))
	return err == nil
}

// SetFFmpegEnv sets the environment of the core command for building and
// running apps with the FFmpeg installed in [ffmpegDir] by core setup
// -video, if any: its pkg-config files, and its shared libraries for the
// apps (an rpath on Linux, and the PATH on Windows, for core run). With
// FFmpeg of Homebrew that is not linked (ffmpeg@9), its pkg-config files.
func SetFFmpegEnv() {
	prepend := func(name, value string) {
		if old := os.Getenv(name); old != "" {
			value += string(os.PathListSeparator) + old
		}
		os.Setenv(name, value)
	}
	if ffmpegInstalled() {
		dir := ffmpegDir()
		prepend("PKG_CONFIG_PATH", filepath.Join(dir, "lib", "pkgconfig"))
		switch runtime.GOOS {
		case "linux":
			lib := filepath.Join(dir, "lib")
			os.Setenv("CGO_LDFLAGS", strings.TrimSpace(os.Getenv("CGO_LDFLAGS")+" -Wl,-rpath,"+lib))
		case "windows":
			prepend("PATH", filepath.Join(dir, "bin"))
		}
		return
	}
	if runtime.GOOS == "darwin" {
		// a versioned formula is keg-only (not linked), so it is not found
		// by pkg-config without its path
		formula := "ffmpeg@" + ffmpegMajor()
		pc := filepath.Join(brewPrefix(), "opt", formula, "lib", "pkgconfig")
		if _, err := os.Stat(pc); err == nil {
			prepend("PKG_CONFIG_PATH", pc)
		}
	}
}

// brewPrefix returns the prefix of Homebrew, /opt/homebrew on Apple
// silicon and /usr/local on Intel.
func brewPrefix() string {
	if runtime.GOARCH == "arm64" {
		return "/opt/homebrew"
	}
	return "/usr/local"
}

// systemFFmpegOK returns whether pkg-config finds FFmpeg of [ffmpegVersion],
// and its version.
func systemFFmpegOK() (bool, string) {
	v, err := exec.Silent().Output("pkg-config", "--modversion", "libavutil")
	if err != nil {
		return false, ""
	}
	v = strings.TrimSpace(v)
	major, _, _ := strings.Cut(v, ".")
	m, err := strconv.Atoi(major)
	return err == nil && m == ffmpegAvutilMajor, v
}

// setupFFmpeg installs FFmpeg for the video package (see ffmpeg.go), or
// updates it if update.
func setupFFmpeg(update bool) error {
	if _, err := exec.LookPath("pkg-config"); err != nil {
		// (go-astiav finds FFmpeg with it; Homebrew installs it with ffmpeg)
		defer logx.PrintlnError("pkg-config is needed to build with FFmpeg; install it with the package manager of the system (as pkg-config or pkgconf)")
	}
	if runtime.GOOS == "darwin" {
		return setupFFmpegBrew(update)
	}
	if ok, v := systemFFmpegOK(); ok && !ffmpegInstalled() {
		logx.PrintlnWarn("FFmpeg " + ffmpegVersion + " is already installed on the system (libavutil " + v + ")")
		return nil
	}
	return setupFFmpegBtbN(update)
}

// setupFFmpegBrew installs or updates FFmpeg with Homebrew.
func setupFFmpegBrew(update bool) error {
	if _, err := exec.LookPath("brew"); err != nil {
		return errors.New("Homebrew is needed to install FFmpeg for video: see https://brew.sh")
	}
	// the ffmpeg formula, if it is of the version, or else the formula of
	// the version (as Homebrew keeps ffmpeg@7 and ffmpeg@8 for older ones)
	formula := "ffmpeg"
	out, err := exec.Silent().Output("brew", "info", "--json=v2", "--formula", "ffmpeg")
	if err != nil {
		return err
	}
	var info struct {
		Formulae []struct {
			Versions struct{ Stable string }
		}
	}
	if err := json.Unmarshal([]byte(out), &info); err != nil || len(info.Formulae) == 0 {
		return fmt.Errorf("reading the version of the ffmpeg formula of Homebrew: %w", err)
	}
	if !strings.HasPrefix(info.Formulae[0].Versions.Stable, ffmpegMajor()+".") {
		formula = "ffmpeg@" + ffmpegMajor()
	}
	vc := exec.Verbose().SetBuffer(false)
	if update {
		logx.PrintlnWarn("updating FFmpeg (" + formula + ") with Homebrew")
		if err := vc.Run("brew", "upgrade", formula); err == nil {
			return nil
		}
	}
	logx.PrintlnWarn("installing FFmpeg (" + formula + ") with Homebrew")
	return vc.Run("brew", "install", formula)
}

// btbnAsset is an asset of the release of BtbN.
type btbnAsset struct {
	Name      string    `json:"name"`
	URL       string    `json:"browser_download_url"`
	UpdatedAt time.Time `json:"updated_at"`
	Size      int64     `json:"size"`
}

// btbnAssetName returns the name of the asset of the prebuilt shared LGPL
// build of BtbN of [ffmpegVersion] for the given platform, or "" if there
// is none.
func btbnAssetName(goos, goarch string) string {
	var plat, ext string
	switch goos + "/" + goarch {
	case "linux/amd64":
		plat, ext = "linux64", "tar.xz"
	case "linux/arm64":
		plat, ext = "linuxarm64", "tar.xz"
	case "windows/amd64":
		plat, ext = "win64", "zip"
	case "windows/arm64":
		plat, ext = "winarm64", "zip"
	default:
		return ""
	}
	return fmt.Sprintf("ffmpeg-n%s-latest-%s-lgpl-shared-%s.%s", ffmpegVersion, plat, ffmpegVersion, ext)
}

// setupFFmpegBtbN installs or updates the prebuilt FFmpeg of BtbN in
// [ffmpegDir]. It is only downloaded again for an update with a newer
// build than the installed one.
func setupFFmpegBtbN(update bool) error {
	name := btbnAssetName(runtime.GOOS, runtime.GOARCH)
	dir := ffmpegDir()
	if name == "" || dir == "" {
		return fmt.Errorf("there is no prebuilt FFmpeg for %s/%s: install the development libraries of FFmpeg %s yourself (found with pkg-config)", runtime.GOOS, runtime.GOARCH, ffmpegVersion)
	}
	stamp := filepath.Join(dir, "core-build.txt") // the asset and its time (see installFFmpegArchive)
	if ffmpegInstalled() && !update {
		logx.PrintlnWarn("FFmpeg " + ffmpegVersion + " is already installed in " + dir)
		return nil
	}
	asset, err := btbnLatest(name)
	if err != nil {
		return err
	}
	want := asset.Name + " " + asset.UpdatedAt.UTC().Format(time.RFC3339)
	if b, err := os.ReadFile(stamp); err == nil && ffmpegInstalled() && strings.TrimSpace(string(b)) == want {
		logx.PrintlnWarn("FFmpeg " + ffmpegVersion + " in " + dir + " is up to date")
		return nil
	}
	logx.PrintlnWarn(fmt.Sprintf("downloading FFmpeg %s (%s, %d MB)", ffmpegVersion, asset.Name, asset.Size>>20))
	if err := os.MkdirAll(filepath.Dir(dir), 0777); err != nil {
		return err
	}
	tmp, err := os.MkdirTemp(filepath.Dir(dir), "ffmpeg-download-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	archive := filepath.Join(tmp, asset.Name)
	if err := download(asset.URL, archive); err != nil {
		return err
	}
	if err := installFFmpegArchive(archive, dir, want); err != nil {
		return err
	}
	logx.PrintlnWarn("installed FFmpeg " + ffmpegVersion + " in " + dir)
	return nil
}

// installFFmpegArchive extracts the given archive of BtbN into the given
// directory of FFmpeg, replacing it, and records the given stamp of its
// build in it. The archive has one top directory (the name of the asset
// without its extension), which becomes the directory. Both GNU tar (with
// xz) and the tar of Windows (bsdtar) extract the archives.
func installFFmpegArchive(archive, dir, stamp string) error {
	tmp, err := os.MkdirTemp(filepath.Dir(dir), "ffmpeg-extract-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	name := filepath.Base(archive)
	if err := exec.Verbose().SetBuffer(false).Run("tar", "-xf", archive, "-C", tmp); err != nil {
		return fmt.Errorf("extracting %s: %w", name, err)
	}
	top := strings.TrimSuffix(strings.TrimSuffix(name, ".zip"), ".tar.xz")
	if _, err := os.Stat(filepath.Join(tmp, top, "lib", "pkgconfig")); err != nil {
		return fmt.Errorf("%s has no %s/lib/pkgconfig", name, top)
	}
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	if err := os.Rename(filepath.Join(tmp, top), dir); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "core-build.txt"), []byte(stamp+"\n"), 0666)
}

// btbnLatest returns the asset of the given name of the latest release
// of BtbN.
func btbnLatest(name string) (*btbnAsset, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(ffmpegReleases)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("getting the release of the FFmpeg builds: %s", resp.Status)
	}
	var rel struct {
		Assets []btbnAsset `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}
	for i := range rel.Assets {
		if rel.Assets[i].Name == name {
			return &rel.Assets[i], nil
		}
	}
	return nil, fmt.Errorf("the FFmpeg builds have no %s", name)
}

// download downloads the given URL to the given file.
func download(url, file string) error {
	client := &http.Client{Timeout: 15 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("downloading %s: %s", url, resp.Status)
	}
	f, err := os.Create(file)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
