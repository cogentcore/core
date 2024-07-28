Now that you have read the [basics](../basics) and [tutorials](../tutorials), you are ready to start developing with Cogent Core on your system. Complete the following steps to install Cogent Core:

1. Download and install Go from [the Go website](https://go.dev/doc/install) if you do not already have Go 1.22+ installed.
2. Run `go install cogentcore.org/core/cmd/core@main` to install the Cogent Core command line tool.
3. Run `core setup` to install platform-specific dependencies.
    * This installs the [xcode-tools](https://mac.install.guide/commandlinetools/4) and [Vulkan](https://vulkan.lunarg.com/sdk/home) on macOS, various window management libraries on Linux, and [TDM-GCC](https://jmeubank.github.io/tdm-gcc/) and [Git](https://git-scm.com/download/win) on Windows.
    * You should run the command from your home directory.
    * You may need to enter your password so that it can run `sudo` commands.
    * You may need to restart your shell after running the command.
    * On Windows, you may need to run the command from a shell/prompt running as administrator (but still in your home directory). You may need to follow steps in installers; the default options should be fine. If it asks you about Git Bash, you should choose to install it, and you should use it as your main shell for development.
