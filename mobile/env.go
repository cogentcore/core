package mobile

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"slices"

	"goki.dev/goki/config"
	"goki.dev/goki/mobile/sdkpath"
	"goki.dev/grog"
	"goki.dev/xe"
)

// General mobile build environment. Initialized by envInit.
var (
	GoMobilePath string                       // $GOPATH/pkg/gomobile
	AndroidEnv   map[string]map[string]string // android arch -> map[string]string
	AppleEnv     map[string]map[string]string // platform/arch -> map[string]string
	AppleNM      string
)

func IsAndroidPlatform(platform string) bool {
	return platform == "android"
}

func IsApplePlatform(platform string) bool {
	return slices.Contains(ApplePlatforms, platform)
}

var ApplePlatforms = []string{"ios", "iossimulator", "macos", "maccatalyst"}

func PlatformArchs(platform string) []string {
	switch platform {
	case "ios":
		return []string{"arm64"}
	case "iossimulator":
		return []string{"arm64", "amd64"}
	case "macos", "maccatalyst":
		return []string{"arm64", "amd64"}
	case "android":
		return []string{"arm", "arm64", "386", "amd64"}
	default:
		panic(fmt.Sprintf("unexpected platform: %s", platform))
	}
}

func IsSupportedArch(platform, arch string) bool {
	return slices.Contains(PlatformArchs(platform), arch)
}

// PlatformOS returns the correct GOOS value for platform.
func PlatformOS(platform string) string {
	switch platform {
	case "android":
		return "android"
	case "ios", "iossimulator":
		return "ios"
	case "macos", "maccatalyst":
		// For "maccatalyst", Go packages should be built with GOOS=darwin,
		// not GOOS=ios, since the underlying OS (and kernel, runtime) is macOS.
		// We also apply a "macos" or "maccatalyst" build tag, respectively.
		// See below for additional context.
		return "darwin"
	default:
		panic(fmt.Sprintf("unexpected platform: %s", platform))
	}
}

func PlatformTags(platform string) []string {
	switch platform {
	case "android":
		return []string{"android"}
	case "ios", "iossimulator":
		return []string{"ios"}
	case "macos":
		return []string{"macos"}
	case "maccatalyst":
		// Mac Catalyst is a subset of iOS APIs made available on macOS
		// designed to ease porting apps developed for iPad to macOS.
		// See https://developer.apple.com/mac-catalyst/.
		// Because of this, when building a Go package targeting maccatalyst,
		// GOOS=darwin (not ios). To bridge the gap and enable maccatalyst
		// packages to be compiled, we also specify the "ios" build tag.
		// To help discriminate between darwin, ios, macos, and maccatalyst
		// targets, there is also a "maccatalyst" tag.
		// Some additional context on this can be found here:
		// https://stackoverflow.com/questions/12132933/preprocessor-macro-for-os-x-targets/49560690#49560690
		// TODO(ydnar): remove tag "ios" when cgo supports Catalyst
		// See golang.org/issues/47228
		return []string{"ios", "macos", "maccatalyst"}
	default:
		panic(fmt.Sprintf("unexpected platform: %s", platform))
	}
}

func BuildEnvInit(c *config.Config) (cleanup func(), err error) {
	// Find gomobilepath.
	gopath := GoEnv("GOPATH")
	for _, p := range filepath.SplitList(gopath) {
		GoMobilePath = filepath.Join(p, "pkg", "gomobile")
		if _, err := os.Stat(GoMobilePath); c.Build.PrintOnly || err == nil {
			break
		}
	}

	grog.PrintlnInfo("GOMOBILE=" + GoMobilePath)

	// Check the toolchain is in a good state.
	// Pick a temporary directory for assembling an apk/app.
	if GoMobilePath == "" {
		return nil, errors.New("toolchain not installed, run `gomobile init`")
	}

	cleanupFn := func() {
		if c.Build.Work {
			fmt.Printf("WORK=%s\n", TmpDir)
			return
		}
		xe.RemoveAll(TmpDir)
	}
	if c.Build.PrintOnly {
		TmpDir = "$WORK"
		cleanupFn = func() {}
	} else {
		TmpDir, err = os.MkdirTemp("", "gomobile-work-")
		if err != nil {
			return nil, err
		}
	}
	grog.PrintlnInfo("WORK=" + TmpDir)

	if err := EnvInit(c); err != nil {
		return nil, err
	}

	return cleanupFn, nil
}

func EnvInit(c *config.Config) (err error) {
	// Setup the cross-compiler environments.
	if ndkRoot, err := NDKRoot(c); err == nil {
		AndroidEnv = make(map[string]map[string]string)
		if c.Build.AndroidMinSDK < MinAndroidSDK {
			return fmt.Errorf("gomobile requires Android API level >= %d", MinAndroidSDK)
		}
		for arch, toolchain := range NDK {
			clang := toolchain.Path(c, ndkRoot, "clang")
			clangpp := toolchain.Path(c, ndkRoot, "clang++")
			if !c.Build.PrintOnly {
				tools := []string{clang, clangpp}
				if runtime.GOOS == "windows" {
					// Because of https://github.com/android-ndk/ndk/issues/920,
					// we require r19c, not just r19b. Fortunately, the clang++.cmd
					// script only exists in r19c.
					tools = append(tools, clangpp+".cmd")
				}
				for _, tool := range tools {
					_, err = os.Stat(tool)
					if err != nil {
						return fmt.Errorf("no compiler for %s was found in the NDK (tried %s). Make sure your NDK version is >= r19c. Use `sdkmanager --update` to update it", arch, tool)
					}
				}
			}
			AndroidEnv[arch] = map[string]string{
				"GOOS":        "android",
				"GOARCH":      arch,
				"CC":          clang,
				"CXX":         clangpp,
				"CGO_ENABLED": "1",
			}
			if arch == "arm" {
				AndroidEnv[arch]["GOARM"] = "7"
			}
		}
	}

	if !XCodeAvailable() {
		return nil
	}

	AppleNM = "nm"
	AppleEnv = make(map[string]map[string]string)
	for _, platform := range ApplePlatforms {
		for _, arch := range PlatformArchs(platform) {
			var goos, sdk, clang, cflags string
			var err error
			switch platform {
			case "ios":
				goos = "ios"
				sdk = "iphoneos"
				clang, cflags, err = EnvClang(c, sdk)
				cflags += " -mios-version-min=" + c.Build.IOSVersion
			case "iossimulator":
				goos = "ios"
				sdk = "iphonesimulator"
				clang, cflags, err = EnvClang(c, sdk)
				cflags += " -mios-simulator-version-min=" + c.Build.IOSVersion
			case "maccatalyst":
				// Mac Catalyst is a subset of iOS APIs made available on macOS
				// designed to ease porting apps developed for iPad to macOS.
				// See https://developer.apple.com/mac-catalyst/.
				// Because of this, when building a Go package targeting maccatalyst,
				// GOOS=darwin (not ios). To bridge the gap and enable maccatalyst
				// packages to be compiled, we also specify the "ios" build tag.
				// To help discriminate between darwin, ios, macos, and maccatalyst
				// targets, there is also a "maccatalyst" tag.
				// Some additional context on this can be found here:
				// https://stackoverflow.com/questions/12132933/preprocessor-macro-for-os-x-targets/49560690#49560690
				goos = "darwin"
				sdk = "macosx"
				clang, cflags, err = EnvClang(c, sdk)
				// TODO(ydnar): the following 3 lines MAY be needed to compile
				// packages or apps for maccatalyst. Commenting them out now in case
				// it turns out they are necessary. Currently none of the example
				// apps will build for macos or maccatalyst because they have a
				// GLKit dependency, which is deprecated on all Apple platforms, and
				// broken on maccatalyst (GLKView isn’t available).
				// sysroot := strings.SplitN(cflags, " ", 2)[1]
				// cflags += " -isystem " + sysroot + "/System/iOSSupport/usr/include"
				// cflags += " -iframework " + sysroot + "/System/iOSSupport/System/Library/Frameworks"
				switch arch {
				case "amd64":
					cflags += " -target x86_64-apple-ios" + c.Build.IOSVersion + "-macabi"
				case "arm64":
					cflags += " -target arm64-apple-ios" + c.Build.IOSVersion + "-macabi"
				}
			case "macos":
				goos = "darwin"
				sdk = "macosx" // Note: the SDK is called "macosx", not "macos"
				clang, cflags, err = EnvClang(c, sdk)
			default:
				panic(fmt.Errorf("unknown Apple target: %s/%s", platform, arch))
			}

			if err != nil {
				return err
			}

			AppleEnv[platform+"/"+arch] = map[string]string{
				"GOOS":         goos,
				"GOARCH":       arch,
				"GOFLAGS":      "-tags=" + strings.Join(PlatformTags(platform), ","),
				"CC":           clang,
				"CXX":          clang + "++",
				"CGO_CFLAGS":   cflags + " -arch " + ArchClang(arch),
				"CGO_CXXFLAGS": cflags + " -arch " + ArchClang(arch),
				"CGO_LDFLAGS":  cflags + " -arch " + ArchClang(arch),
				"CGO_ENABLED":  "1",
				"DARWIN_SDK":   sdk,
			}
		}
	}

	return nil
}

// ABI maps GOARCH values to Android ABI strings.
// See https://developer.android.com/ndk/guides/abis
func ABI(goarch string) string {
	switch goarch {
	case "arm":
		return "armeabi-v7a"
	case "arm64":
		return "arm64-v8a"
	case "386":
		return "x86"
	case "amd64":
		return "x86_64"
	default:
		return ""
	}
}

// CheckNDKRoot returns nil if the NDK in `ndkRoot` supports the current configured
// API version and all the specified Android targets.
func CheckNDKRoot(c *config.Config, ndkRoot string, targets []config.Platform) error {
	platformsJson, err := os.Open(filepath.Join(ndkRoot, "meta", "platforms.json"))
	if err != nil {
		return err
	}
	defer platformsJson.Close()
	decoder := json.NewDecoder(platformsJson)
	supportedVersions := struct {
		Min int
		Max int
	}{}
	if err := decoder.Decode(&supportedVersions); err != nil {
		return err
	}
	if supportedVersions.Min > c.Build.AndroidMinSDK ||
		supportedVersions.Max < c.Build.AndroidMinSDK {
		return fmt.Errorf("unsupported API version %d (not in %d..%d)", c.Build.AndroidMinSDK, supportedVersions.Min, supportedVersions.Max)
	}
	abisJson, err := os.Open(filepath.Join(ndkRoot, "meta", "abis.json"))
	if err != nil {
		return err
	}
	defer abisJson.Close()
	decoder = json.NewDecoder(abisJson)
	abis := make(map[string]struct{})
	if err := decoder.Decode(&abis); err != nil {
		return err
	}
	for _, target := range targets {
		if !IsAndroidPlatform(target.OS) {
			continue
		}
		if _, found := abis[ABI(target.Arch)]; !found {
			return fmt.Errorf("ndk does not support %s", target.OS)
		}
	}
	return nil
}

// CompatibleNDKRoots searches the side-by-side NDK dirs for compatible SDKs.
func CompatibleNDKRoots(c *config.Config, ndkForest string, targets []config.Platform) ([]string, error) {
	ndkDirs, err := ioutil.ReadDir(ndkForest)
	if err != nil {
		return nil, err
	}
	compatibleNDKRoots := []string{}
	var lastErr error
	for _, dirent := range ndkDirs {
		ndkRoot := filepath.Join(ndkForest, dirent.Name())
		lastErr = CheckNDKRoot(c, ndkRoot, targets)
		if lastErr == nil {
			compatibleNDKRoots = append(compatibleNDKRoots, ndkRoot)
		}
	}
	if len(compatibleNDKRoots) > 0 {
		return compatibleNDKRoots, nil
	}
	return nil, lastErr
}

// NDKVersion returns the full version number of an installed copy of the NDK,
// or "" if it cannot be determined.
func NDKVersion(ndkRoot string) string {
	properties, err := os.Open(filepath.Join(ndkRoot, "source.properties"))
	if err != nil {
		return ""
	}
	defer properties.Close()
	// Parse the version number out of the .properties file.
	// See https://en.wikipedia.org/wiki/.properties
	scanner := bufio.NewScanner(properties)
	for scanner.Scan() {
		line := scanner.Text()
		tokens := strings.SplitN(line, "=", 2)
		if len(tokens) != 2 {
			continue
		}
		if strings.TrimSpace(tokens[0]) == "Pkg.Revision" {
			return strings.TrimSpace(tokens[1])
		}
	}
	return ""
}

// NDKRoot returns the root path of an installed NDK that supports all the
// specified Android targets. For details of NDK locations, see
// https://github.com/android/ndk-samples/wiki/Configure-NDK-Path
func NDKRoot(c *config.Config, targets ...config.Platform) (string, error) {
	if c.Build.PrintOnly {
		return "$NDK_PATH", nil
	}

	// Try the ANDROID_NDK_HOME variable.  This approach is deprecated, but it
	// has the highest priority because it represents an explicit user choice.
	if ndkRoot := os.Getenv("ANDROID_NDK_HOME"); ndkRoot != "" {
		if err := CheckNDKRoot(c, ndkRoot, targets); err != nil {
			return "", fmt.Errorf("ANDROID_NDK_HOME specifies %s, which is unusable: %w", ndkRoot, err)
		}
		return ndkRoot, nil
	}

	androidHome, err := sdkpath.AndroidHome()
	if err != nil {
		return "", fmt.Errorf("could not locate Android SDK: %w", err)
	}

	// Use the newest compatible NDK under the side-by-side path arrangement.
	ndkForest := filepath.Join(androidHome, "ndk")
	ndkRoots, sideBySideErr := CompatibleNDKRoots(c, ndkForest, targets)
	if len(ndkRoots) != 0 {
		// Choose the latest version that supports the build configuration.
		// NDKs whose version cannot be determined will be least preferred.
		// In the event of a tie, the later ndkRoot will win.
		maxVersion := ""
		var selected string
		for _, ndkRoot := range ndkRoots {
			version := NDKVersion(ndkRoot)
			if version >= maxVersion {
				maxVersion = version
				selected = ndkRoot
			}
		}
		return selected, nil
	}
	// Try the deprecated NDK location.
	ndkRoot := filepath.Join(androidHome, "ndk-bundle")
	if legacyErr := CheckNDKRoot(c, ndkRoot, targets); legacyErr != nil {
		return "", fmt.Errorf("no usable NDK in %s: %w, %v", androidHome, sideBySideErr, legacyErr)
	}
	return ndkRoot, nil
}

func EnvClang(c *config.Config, sdkName string) (clang, cflags string, err error) {
	if c.Build.PrintOnly {
		return sdkName + "-clang", "-isysroot " + sdkName, nil
	}
	out, err := xe.Minor().Output("xcrun", "--sdk", sdkName, "--find", "clang")
	if err != nil {
		return "", "", fmt.Errorf("xcrun --find: %v\n%s", err, out)
	}
	clang = strings.TrimSpace(string(out))

	out, err = xe.Minor().Output("xcrun", "--sdk", sdkName, "--show-sdk-path")
	if err != nil {
		return "", "", fmt.Errorf("xcrun --show-sdk-path: %v\n%s", err, out)
	}
	sdk := strings.TrimSpace(string(out))
	return clang, "-isysroot " + sdk, nil
}

func ArchClang(goarch string) string {
	switch goarch {
	case "arm":
		return "armv7"
	case "arm64":
		return "arm64"
	case "386":
		return "i386"
	case "amd64":
		return "x86_64"
	default:
		panic(fmt.Sprintf("unknown GOARCH: %q", goarch))
	}
}

func ArchNDK() string {
	if runtime.GOOS == "windows" && runtime.GOARCH == "386" {
		return "windows"
	} else {
		var arch string
		switch runtime.GOARCH {
		case "386":
			arch = "x86"
		case "amd64":
			arch = "x86_64"
		case "arm64":
			// Android NDK does not contain arm64 toolchains (until and
			// including NDK 23), use use x86_64 instead. See:
			// https://github.com/android/ndk/issues/1299
			if runtime.GOOS == "darwin" {
				arch = "x86_64"
				break
			}
			fallthrough
		default:
			panic("unsupported GOARCH: " + runtime.GOARCH)
		}
		return runtime.GOOS + "-" + arch
	}
}

type NDKToolchain struct {
	Arch           string
	ABI            string
	MinAPI         int
	ToolPrefix     string
	ClangPrefixVal string // ClangPrefix is taken by a method
}

func (tc *NDKToolchain) ClangPrefix(c *config.Config) string {
	if c.Build.AndroidMinSDK < tc.MinAPI {
		return fmt.Sprintf("%s%d", tc.ClangPrefixVal, tc.MinAPI)
	}
	return fmt.Sprintf("%s%d", tc.ClangPrefixVal, c.Build.AndroidMinSDK)
}

func (tc *NDKToolchain) Path(c *config.Config, ndkRoot, toolName string) string {
	cmdFromPref := func(pref string) string {
		return filepath.Join(ndkRoot, "toolchains", "llvm", "prebuilt", ArchNDK(), "bin", pref+"-"+toolName)
	}

	var cmd string
	switch toolName {
	case "clang", "clang++":
		cmd = cmdFromPref(tc.ClangPrefix(c))
	default:
		cmd = cmdFromPref(tc.ToolPrefix)
		// Starting from NDK 23, GNU binutils are fully migrated to LLVM binutils.
		// See https://android.googlesource.com/platform/ndk/+/master/docs/Roadmap.md#ndk-r23
		if _, err := os.Stat(cmd); errors.Is(err, fs.ErrNotExist) {
			cmd = cmdFromPref("llvm")
		}
	}
	return cmd
}

type NDKConfig map[string]NDKToolchain // map: GOOS->androidConfig.

func (nc NDKConfig) Toolchain(arch string) NDKToolchain {
	tc, ok := nc[arch]
	if !ok {
		panic(`unsupported architecture: ` + arch)
	}
	return tc
}

var NDK = NDKConfig{
	"arm": {
		Arch:           "arm",
		ABI:            "armeabi-v7a",
		MinAPI:         16,
		ToolPrefix:     "arm-linux-androideabi",
		ClangPrefixVal: "armv7a-linux-androideabi",
	},
	"arm64": {
		Arch:           "arm64",
		ABI:            "arm64-v8a",
		MinAPI:         21,
		ToolPrefix:     "aarch64-linux-android",
		ClangPrefixVal: "aarch64-linux-android",
	},

	"386": {
		Arch:           "x86",
		ABI:            "x86",
		MinAPI:         16,
		ToolPrefix:     "i686-linux-android",
		ClangPrefixVal: "i686-linux-android",
	},
	"amd64": {
		Arch:           "x86_64",
		ABI:            "x86_64",
		MinAPI:         21,
		ToolPrefix:     "x86_64-linux-android",
		ClangPrefixVal: "x86_64-linux-android",
	},
}

func XCodeAvailable() bool {
	err := exec.Command("xcrun", "xcodebuild", "-version").Run()
	return err == nil
}
