// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mobile

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	cexec "cogentcore.org/core/base/exec"
	"cogentcore.org/core/base/logx"
	"cogentcore.org/core/cmd/mobile/sdkpath"
)

// The Android emulator: core setup -androidsim installs the Android SDK
// (its command line tools, platform tools, emulator, an NDK, and a system
// image of [androidSimAPI] for the architecture of the system) with the
// sdkmanager of the SDK, and makes an Android virtual device (AVD) named
// [AndroidSimAVD]. core run android then runs apps on a connected Android
// device, or else boots the emulator (see [AndroidDevice]).

const (
	// androidRepo is the URL of the repository of the packages of the
	// Android SDK, whose index ([androidRepoIndex]) has their archives.
	androidRepo = "https://dl.google.com/android/repository/"

	// androidRepoIndex is the index of the packages of the Android SDK.
	androidRepoIndex = androidRepo + "repository2-3.xml"

	// androidSimAPI is the Android API level of the system image of the
	// emulator (a stable one, with the Google APIs).
	androidSimAPI = 36

	// AndroidSimAVD is the name of the Android virtual device of the
	// emulator that core setup -androidsim makes.
	AndroidSimAVD = "Cogent"
)

// androidSimDevices are the hardware profiles for [AndroidSimAVD], in order
// of preference: the first one that avdmanager has (older versions of it
// do not have the newer ones).
var androidSimDevices = []string{"pixel_9", "pixel_8", "pixel_7", "pixel_6", "pixel_5", "pixel_4", "pixel"}

// AndroidTool returns the path of the given tool of the Android SDK (adb,
// emulator, sdkmanager, or avdmanager), or else just its name, to be found
// on the path.
func AndroidTool(name string) string {
	sdk, err := sdkpath.AndroidHomePath()
	if err != nil {
		return name
	}
	exe := ""
	if runtime.GOOS == "windows" {
		exe = ".exe"
	}
	var path string
	switch name {
	case "adb":
		path = filepath.Join(sdk, "platform-tools", "adb"+exe)
	case "emulator":
		path = filepath.Join(sdk, "emulator", "emulator"+exe)
	default:
		if runtime.GOOS == "windows" {
			exe = ".bat"
		}
		path = filepath.Join(sdk, "cmdline-tools", "latest", "bin", name+exe)
	}
	if _, err := os.Stat(path); err != nil {
		return name
	}
	return path
}

// androidSimImage returns the sdkmanager package of the system image of
// the emulator, for the architecture of the system.
func androidSimImage() string {
	abi := "x86_64"
	if runtime.GOARCH == "arm64" {
		abi = "arm64-v8a"
	}
	return fmt.Sprintf("system-images;android-%d;google_apis;%s", androidSimAPI, abi)
}

// SetupAndroidSim installs the Android SDK and makes the AVD of the
// emulator (see the comment above [AndroidSimAVD]): the command line tools
// from [androidRepo] if they are not installed, which need Java (a JDK,
// which it installs with Homebrew on macOS if there is none), and then the
// rest with their sdkmanager, accepting their licenses. It only installs
// an NDK if there is none.
func SetupAndroidSim() error {
	sdk, err := sdkpath.AndroidHomePath()
	if err != nil {
		return err
	}
	logx.PrintlnWarn("setting up the Android SDK and emulator in " + sdk)
	if err := os.MkdirAll(sdk, 0755); err != nil {
		return err
	}
	vc := cexec.Verbose().SetBuffer(false).SetEnv("ANDROID_HOME", sdk)
	if err := setJavaHome(vc); err != nil {
		return err
	}
	var pkgs []repoPackage // (the index, if needed)
	index := func() ([]repoPackage, error) {
		if pkgs == nil {
			pkgs, err = androidRepoPackages()
		}
		return pkgs, err
	}

	// the command line tools, with sdkmanager
	if AndroidTool("sdkmanager") == "sdkmanager" {
		ps, err := index()
		if err != nil {
			return err
		}
		if err := installCmdlineTools(sdk, ps); err != nil {
			return fmt.Errorf("installing the Android command line tools: %w", err)
		}
	}
	sm := AndroidTool("sdkmanager")
	yes := strings.Repeat("y\n", 100) // (accepts all of the licenses)
	logx.PrintlnWarn("accepting the licenses of the Android SDK")
	lc := cexec.Silent().SetStdin(strings.NewReader(yes))
	lc.Env = vc.Env // (ANDROID_HOME and JAVA_HOME)
	if _, err := lc.Output(sm, "--licenses"); err != nil {
		return fmt.Errorf("accepting the licenses of the Android SDK: %w", err)
	}

	// the packages: the platform tools (adb), the emulator, its system
	// image, and an NDK, for building apps, if there is none
	install := []string{"cmdline-tools;latest", "platform-tools", "emulator", androidSimImage()}
	if !hasNDK(sdk) {
		ps, err := index()
		if err != nil {
			return err
		}
		ndk := latestStable(ps, "ndk;")
		if ndk == "" {
			return errors.New("no stable NDK in the index of the Android SDK")
		}
		install = append(install, ndk)
	}
	if err := vc.SetStdin(strings.NewReader(yes)).Run(sm, append([]string{"--install"}, install...)...); err != nil {
		return fmt.Errorf("installing %s: %w", strings.Join(install, ", "), err)
	}
	return makeAndroidSimAVD(vc)
}

// setJavaHome sets JAVA_HOME in the given config for the tools of the
// Android SDK, which need Java 17 or later, if java does not run as it is:
// on macOS, that of java_home, or else that of the openjdk of Homebrew,
// which it installs if needed.
func setJavaHome(vc *cexec.Config) error {
	if os.Getenv("JAVA_HOME") != "" {
		return nil
	}
	if _, err := cexec.Silent().Output("java", "-version"); err == nil {
		return nil
	}
	notFound := errors.New("the tools of the Android SDK need Java (a JDK of version 17 or later), which is not installed: install it (as the openjdk package of the system), and run this again")
	if runtime.GOOS != "darwin" {
		return notFound
	}
	if home, err := cexec.Silent().Output("/usr/libexec/java_home"); err == nil && strings.TrimSpace(home) != "" {
		vc.SetEnv("JAVA_HOME", strings.TrimSpace(home))
		return nil
	}
	if _, err := exec.LookPath("brew"); err != nil {
		return notFound
	}
	if err := vc.Run("brew", "install", "openjdk"); err != nil {
		return fmt.Errorf("installing Java (openjdk) with Homebrew: %w", err)
	}
	prefix, err := cexec.Silent().Output("brew", "--prefix")
	if err != nil {
		return err
	}
	vc.SetEnv("JAVA_HOME", filepath.Join(strings.TrimSpace(prefix), "opt", "openjdk", "libexec", "openjdk.jdk", "Contents", "Home"))
	return nil
}

// hasNDK returns whether the given Android SDK has an NDK.
func hasNDK(sdk string) bool {
	ndks, _ := filepath.Glob(filepath.Join(sdk, "ndk", "*", "source.properties"))
	_, err := os.Stat(filepath.Join(sdk, "ndk-bundle", "source.properties"))
	return len(ndks) > 0 || err == nil
}

// makeAndroidSimAVD makes [AndroidSimAVD], if there is none, with the
// avdmanager of the SDK, with the keyboard of the computer enabled for it.
func makeAndroidSimAVD(vc *cexec.Config) error {
	if slices.Contains(androidAVDs(), AndroidSimAVD) {
		logx.PrintlnWarn("the Android virtual device " + AndroidSimAVD + " already exists")
		return nil
	}
	am := AndroidTool("avdmanager")
	out, err := cexec.Silent().Output(am, "list", "device", "-c")
	if err != nil {
		return fmt.Errorf("listing the hardware profiles of avdmanager: %w", err)
	}
	device := androidSimDevice(strings.Fields(out))
	// (no to a custom hardware profile)
	err = vc.SetStdin(strings.NewReader("no\n")).Run(am, "create", "avd",
		"-n", AndroidSimAVD, "-k", androidSimImage(), "-d", device)
	if err != nil {
		return fmt.Errorf("making the Android virtual device %s: %w", AndroidSimAVD, err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	ini := filepath.Join(home, ".android", "avd", AndroidSimAVD+".avd", "config.ini")
	if avdHome := os.Getenv("ANDROID_AVD_HOME"); avdHome != "" {
		ini = filepath.Join(avdHome, AndroidSimAVD+".avd", "config.ini")
	}
	b, err := os.ReadFile(ini)
	if err != nil {
		return err
	}
	return os.WriteFile(ini, []byte(setIniValue(string(b), "hw.keyboard", "yes")), 0644)
}

// setIniValue returns the given ini file (key=value lines, as the
// config.ini of an AVD) with the given key set to the given value.
func setIniValue(ini, key, value string) string {
	lines := strings.Split(strings.TrimSuffix(ini, "\n"), "\n")
	set := false
	for i, l := range lines {
		if k, _, ok := strings.Cut(l, "="); ok && strings.TrimSpace(k) == key {
			lines[i] = key + "=" + value
			set = true
		}
	}
	if !set {
		lines = append(lines, key+"="+value)
	}
	return strings.Join(lines, "\n") + "\n"
}

// androidSimDevice returns the hardware profile of [AndroidSimAVD] from the
// given ones of avdmanager: the first one of [androidSimDevices] that it
// has, or else the last pixel one (the newest), or else the first one.
func androidSimDevice(devices []string) string {
	for _, d := range androidSimDevices {
		if slices.Contains(devices, d) {
			return d
		}
	}
	for _, d := range slices.Backward(devices) {
		if strings.HasPrefix(d, "pixel") {
			return d
		}
	}
	if len(devices) > 0 {
		return devices[0]
	}
	return "pixel"
}

// androidAVDs returns the names of the Android virtual devices.
func androidAVDs() []string {
	out, err := cexec.Silent().Output(AndroidTool("emulator"), "-list-avds")
	if err != nil {
		return nil
	}
	var avds []string
	for _, l := range strings.Split(out, "\n") {
		l = strings.TrimSpace(l)
		// (it can log about its setup, as INFO lines)
		if l != "" && !strings.Contains(l, " ") {
			avds = append(avds, l)
		}
	}
	return avds
}

// repoPackage is a package of the index of the Android SDK.
type repoPackage struct {
	Path    string `xml:"path,attr"`
	Channel struct {
		Ref string `xml:"ref,attr"`
	} `xml:"channelRef"`
	Archives []struct {
		URL      string `xml:"complete>url"`
		HostOS   string `xml:"host-os"`
		HostArch string `xml:"host-arch"`
	} `xml:"archives>archive"`
}

// androidRepoPackages returns the packages of the index of the Android SDK.
func androidRepoPackages() ([]repoPackage, error) {
	client := &http.Client{Timeout: time.Minute}
	resp, err := client.Get(androidRepoIndex)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("getting the index of the Android SDK: %s", resp.Status)
	}
	return parseRepoPackages(resp.Body)
}

// parseRepoPackages returns the packages of the given index of the
// Android SDK.
func parseRepoPackages(r io.Reader) ([]repoPackage, error) {
	var repo struct {
		Packages []repoPackage `xml:"remotePackage"`
	}
	if err := xml.NewDecoder(r).Decode(&repo); err != nil {
		return nil, fmt.Errorf("reading the index of the Android SDK: %w", err)
	}
	return repo.Packages, nil
}

// latestStable returns the path of the latest stable package with the
// given prefix of its path and a version after it (as ndk;29.0.14206865).
func latestStable(pkgs []repoPackage, prefix string) string {
	best, bestV := "", []int(nil)
	for _, p := range pkgs {
		if p.Channel.Ref != "channel-0" || !strings.HasPrefix(p.Path, prefix) {
			continue
		}
		v := versionNumbers(strings.TrimPrefix(p.Path, prefix))
		if v == nil {
			continue
		}
		if best == "" || slices.Compare(v, bestV) > 0 {
			best, bestV = p.Path, v
		}
	}
	return best
}

// versionNumbers returns the numbers of the given dotted version, or nil
// if it is not one.
func versionNumbers(v string) []int {
	var nums []int
	for s := range strings.SplitSeq(v, ".") {
		n, err := strconv.Atoi(s)
		if err != nil {
			return nil
		}
		nums = append(nums, n)
	}
	return nums
}

// archiveURL returns the URL of the archive of the given package of the
// index for the given system (GOOS and GOARCH), or "" if there is none.
func archiveURL(pkgs []repoPackage, path, goos, goarch string) string {
	hostOS := map[string]string{"darwin": "macosx", "linux": "linux", "windows": "windows"}[goos]
	hostArch := map[string]string{"arm64": "aarch64", "amd64": "x64"}[goarch]
	for _, p := range pkgs {
		if p.Path != path {
			continue
		}
		for _, a := range p.Archives {
			if a.HostOS == hostOS && (a.HostArch == "" || a.HostArch == hostArch) {
				return androidRepo + a.URL
			}
		}
	}
	return ""
}

// installCmdlineTools installs the latest command line tools of the
// Android SDK from the given index into the given SDK, as
// cmdline-tools/latest, where sdkmanager updates them.
func installCmdlineTools(sdk string, pkgs []repoPackage) error {
	url := archiveURL(pkgs, "cmdline-tools;latest", runtime.GOOS, runtime.GOARCH)
	if url == "" {
		return fmt.Errorf("no command line tools for %s/%s in the index", runtime.GOOS, runtime.GOARCH)
	}
	tmp, err := os.MkdirTemp(sdk, "cmdline-tools-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	zip := filepath.Join(tmp, filepath.Base(url))
	logx.PrintlnWarn("downloading " + url)
	if err := downloadFile(url, zip); err != nil {
		return err
	}
	// (the zip has a cmdline-tools directory; tar extracts zips on all
	// of the platforms, as bsdtar on macOS and Windows)
	if err := cexec.Verbose().SetBuffer(false).Run("tar", "-xf", zip, "-C", tmp); err != nil {
		return fmt.Errorf("extracting %s: %w", filepath.Base(zip), err)
	}
	dir := filepath.Join(sdk, "cmdline-tools")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	latest := filepath.Join(dir, "latest")
	if err := os.RemoveAll(latest); err != nil {
		return err
	}
	return os.Rename(filepath.Join(tmp, "cmdline-tools"), latest)
}

// downloadFile downloads the given URL to the given file.
func downloadFile(url, file string) error {
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

// androidDevice is a device of adb devices.
type androidDevice struct {
	serial, state string
}

// emulator returns whether the device is an emulator.
func (d androidDevice) emulator() bool {
	return strings.HasPrefix(d.serial, "emulator-")
}

// androidDevices returns the devices of adb devices.
func androidDevices() ([]androidDevice, error) {
	out, err := cexec.Silent().Output(AndroidTool("adb"), "devices")
	if err != nil {
		return nil, err
	}
	return parseAndroidDevices(out), nil
}

// parseAndroidDevices returns the devices of the given output of adb
// devices.
func parseAndroidDevices(out string) []androidDevice {
	var devs []androidDevice
	for _, l := range strings.Split(out, "\n") {
		f := strings.Fields(l)
		if len(f) == 2 && f[0] != "List" {
			devs = append(devs, androidDevice{f[0], f[1]})
		}
	}
	return devs
}

// AndroidDevice returns the serial of the Android device to run apps on:
// that of ANDROID_SERIAL, or else a connected device, preferring a phone
// (not an emulator) for when a phone is plugged in while the emulator is
// running, or else the emulator, which it boots with [AndroidSimAVD] (or
// the first AVD, if there is no [AndroidSimAVD]; see core setup
// -androidsim), waiting for it to finish booting.
func AndroidDevice() (string, error) {
	if s := os.Getenv("ANDROID_SERIAL"); s != "" {
		return s, nil
	}
	devs, err := androidDevices()
	if err != nil {
		return "", fmt.Errorf("listing the Android devices with adb (see core setup -androidsim): %w", err)
	}
	if s := chooseAndroidDevice(devs); s != "" {
		return s, nil
	}
	return bootAndroidSim()
}

// chooseAndroidDevice returns the serial of the device of the given ones
// to run apps on, preferring a phone to an emulator, or "" if none is
// ready.
func chooseAndroidDevice(devs []androidDevice) string {
	var ready []androidDevice
	for _, d := range devs {
		switch d.state {
		case "device":
			ready = append(ready, d)
		case "unauthorized":
			logx.PrintlnWarn("the Android device " + d.serial + " is unauthorized: allow USB debugging on it")
		}
	}
	slices.SortStableFunc(ready, func(a, b androidDevice) int {
		if a.emulator() == b.emulator() {
			return 0
		} else if a.emulator() {
			return 1
		}
		return -1
	})
	if len(ready) == 0 {
		return ""
	}
	if len(ready) > 1 {
		logx.PrintlnWarn("running on the Android device " + ready[0].serial + "; set ANDROID_SERIAL to choose another one")
	}
	return ready[0].serial
}

// bootAndroidSim boots the emulator with [AndroidSimAVD] (or the first AVD,
// if there is no [AndroidSimAVD]), and returns its serial, once it has
// finished booting. The emulator keeps running.
func bootAndroidSim() (string, error) {
	avds := androidAVDs()
	if len(avds) == 0 {
		return "", errors.New("no Android device is connected, and there is no Android virtual device for the emulator: run core setup -androidsim to make one")
	}
	avd := avds[0]
	if slices.Contains(avds, AndroidSimAVD) {
		avd = AndroidSimAVD
	}
	logx.PrintlnWarn("booting the Android emulator with " + avd)
	cmd := exec.Command(AndroidTool("emulator"), "-avd", avd)
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("starting the Android emulator: %w", err)
	}
	go cmd.Wait() // (it keeps running after this process exits)
	serial := ""
	deadline := time.Now().Add(3 * time.Minute)
	for time.Now().Before(deadline) {
		time.Sleep(time.Second)
		if serial == "" {
			devs, _ := androidDevices()
			for _, d := range devs {
				if d.emulator() {
					serial = d.serial
				}
			}
			continue
		}
		out, err := cexec.Silent().Output(AndroidTool("adb"), "-s", serial, "shell", "getprop", "sys.boot_completed")
		if err == nil && strings.TrimSpace(out) == "1" {
			return serial, nil
		}
	}
	return "", errors.New("the Android emulator did not finish booting in 3 minutes")
}

// AndroidDeviceArch returns the Go architecture of the given Android
// device (from its primary ABI), or "" if it is not known.
func AndroidDeviceArch(serial string) string {
	out, err := cexec.Silent().Output(AndroidTool("adb"), "-s", serial, "shell", "getprop", "ro.product.cpu.abi")
	if err != nil {
		return ""
	}
	return map[string]string{"arm64-v8a": "arm64", "armeabi-v7a": "arm", "x86_64": "amd64", "x86": "386"}[strings.TrimSpace(out)]
}
