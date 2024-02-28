# Installing goki

On all platforms, you need to download and install Go from [the Go website](https://go.dev/doc/install) if you do not already have Go 1.21+ installed. Then, run `go install cogentcore.org/core/core@main` to install the Cogent Core command line tool.

## MacOS

1. Install the xcode command-line tools if you don't already have them by running `xcode-select --install`
2. If you don't already have the Vulkan SDK installed, install it by doing the following:
    * Run `curl -O https://sdk.lunarg.com/sdk/download/1.3.261.1/mac/vulkansdk-macos-1.3.261.1.dmg`
    * Run `open vulkansdk-macos-1.3.261.1.dmg`
    * Double click `InstallVulkan.app`
    * Follow the installation prompts and ignore all warnings about the Vulkan Portability Enumeration extension

## Windows

1. Download and install Git for Windows from the [git website](https://git-scm.com/download/win) if you don't already have it. You should install Git Bash as part of this process and use it for development.
2. Download and install TDM-GCC from [this website](https://jmeubank.github.io/tdm-gcc/)
3. Open Windows Command Prompt and run `cd C:\TDM-GCC-64`
4. Then, run `mingwvars.bat`

## Linux

* If you are on Ubuntu or Debian, run `sudo apt-get install libgl1-mesa-dev xorg-dev`
* If you are on CentOS or Fedora, run `sudo dnf install libX11-devel libXcursor-devel libXrandr-devel libXinerama-devel mesa-libGL-devel libXi-devel libXxf86vm-devel`
