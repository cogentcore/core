# Building Cogent Core Apps

The file(s) produced by `core build` are `.exe` on Windows, `.app` on iOS, `.apk` on Android, `.wasm` and associated `.html` and `.js` files on Web, and no extension on macOS and Linux. The file(s) are placed in the current directory for desktop platforms, and `bin/{platform}` for other platforms (eg: `bin/android`).

For desktop platforms, `core run` just runs the executable directly. For mobile platforms, it installs the executable package on the target device (which can be a simulator or a real connected device) using `core install` and then starts it. For web, it serves the files in `bin/web` at http://localhost:8080.

On mobile and web platforms, `core pack` is the same as `core build`, as that already makes a package on those platforms. On macOS, it makes a `.app` bundle file and a `.dmg` disk image installer file. On Windows, it makes a `.exe` installer file. On Linux, it makes a `.deb` package file.

For macOS, `core install` means running `core pack` and then copying the resulting `.app` file to `Applications`. (TODO: install on Linux and Windows). For mobile platforms, it installs the executable package on the target device, which can be a simulator or a real connected device. For web, it is not applicable.