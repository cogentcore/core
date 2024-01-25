# Building Cogent Core Apps

All Cogent Core apps can be built using normal `go` command line tools (`go build`, `go run`, `go install`, etc). However, the Cogent Core command line tool `core` provides a build process optimized for cross-platform Cogent Core development. This tool automatically sets appropriate linker flags to reduce binary sizes, optimizes binaries for running as standalone GUI apps, and provides support for building for mobile and web platforms.

The `core` command line tool provides four main building commands:

1. `core build` builds a standalone binary executable for the app (`.exe` on Windows, `.app` on iOS, `.apk` on Android, `.wasm` and associated `.html` and `.js` files on Web, and no extension on macOS and Linux). The resulting file(s) are placed in `.core/bin/{platform}` (eg: `.core/bin/android`).

2. `core run` does `core build` and then runs the resulting executable on the target device. For desktop platforms, this means just running the executable directly. For mobile platforms, this means installing the executable package on the target device (which can be a simulator or a real connected device) and then starting it.

3. `core pack` packages the app into a self-contained package and builds an installer for it if applicable. On mobile and web platforms, this is the same as `core build`, as that already makes a package on those platforms. On macOS, this makes a `.app` bundle file and a `.dmg` disk image installer file. On Windows, this makes a `.exe` installer file. On Linux, this makes a `.deb` package file.

4. `core install` installs the app to the target system. For macOS, this means running `core pack` and then copying the resulting `.app` file to `Applications`. For mobile platforms, this installs the executable package on the target device. For web, this is not applicable.