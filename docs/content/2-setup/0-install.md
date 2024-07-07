# Installing Cogent Core

Now that you have read the [basics](../basics) and [tutorials](../tutorials), you are ready to start developing with Cogent Core on your system. For all platforms, complete the following steps to install Cogent Core:

1. Download and install Go from [the Go website](https://go.dev/doc/install) if you do not already have Go 1.22+ installed.
2. Run `go install cogentcore.org/core/cmd/core@main` to install the Cogent Core command line tool.
3. Run `core setup` to install platform-specific dependencies. You may need to enter your password so that it can run `sudo` commands.

## Windows

On Windows, you must also complete the following additional steps:

1. Download and install Git for Windows from the [git website](https://git-scm.com/download/win) if you don't already have it. You should install Git Bash as part of this process and use it for development.
2. Download and install TDM-GCC from [this website](https://jmeubank.github.io/tdm-gcc/)
3. Open Windows Command Prompt and run `cd C:\TDM-GCC-64`
4. Then, run `mingwvars.bat`
