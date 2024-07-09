# Building Cogent Core apps

All Cogent Core apps can be built using normal `go` commands (`go build`, `go run`, `go install`, etc). This means that things such as tests, debuggers, and race detectors work as usual.

However, the Cogent Core command line tool `core` provides a build process optimized for cross-platform Cogent Core development. This tool automatically sets appropriate linker flags to reduce binary sizes, optimizes binaries for running as standalone GUI apps, and provides support for building for mobile and web platforms.

The `core` command line tool provides four main build commands, as documented below. In general, you should use `core run` during development.

1. `core build` builds a standalone binary executable for the app

2. `core run` builds a standalone binary executable for the app and then runs it on the target device

3. `core pack` packages the app into a self-contained package and builds an installer for it

4. `core install` installs the app on the target system

You can build for mobile and web platforms by adding the platform name after the command. For example, you can run `core run android` to build and run an app on an Android device, `core build ios` to build an app for an iOS device, and `core run web` to serve an app on the web.

You can see the [advanced building page](../advanced/building) for more information if you need it.
