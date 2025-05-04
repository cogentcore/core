You can complete the three steps below to **install** Cogent Core on your system. You can also explore the [[#demo]], [[basics]], [[tutorials]], and [[widgets]]. You can also see how to [[build]] apps.

1. Download and install Go from [the Go website](https://go.dev/doc/install) if you do not already have Go 1.23.4+ installed.

2. Run the following command to install the Cogent Core command line tool. You should run this command again each time you update to a new version of Cogent Core.

```sh
go install cogentcore.org/core/cmd/core@main
```

3. **After** reading the info for your platform below, run the following command to install platform-specific dependencies. You **must** restart your shell/prompt/terminal after running the command.

```sh
core setup
```

## macOS

This installs the [xcode-tools](https://mac.install.guide/commandlinetools/4). You may need to follow installer prompts.

## Windows

This installs [w64devkit](https://github.com/skeeto/w64devkit) and [Git](https://git-scm.com/download/win). You **must** run the setup command from a shell/prompt/terminal (such as Command Prompt or Git Bash) explicitly ["Run as administrator"](https://www.howtogeek.com/194041/how-to-open-the-command-prompt-as-administrator-in-windows-10/). You may need to follow steps in installers; the default options are good. After running this command, you **must** run all future commands related to Cogent Core in Git Bash.

## Linux

This installs Go, gcc, and various graphics libraries. You may need to enter your password so that it can run `sudo` commands.

If you want to install manually instead, the commands for each distro are listed below.

<!-- To update this, copy the output of [cogentcore.org/core/cmd/core/cmd.TestLinuxDistroString]; DO NOT EDIT manually -->

{collapsed="true"}
```sh
Debian/Ubuntu:  sudo apt install gcc libgl1-mesa-dev libegl1-mesa-dev mesa-vulkan-drivers xorg-dev
Fedora:         sudo dnf install gcc libX11-devel libXcursor-devel libXrandr-devel libXinerama-devel mesa-libGL-devel libXi-devel libXxf86vm-devel mesa-vulkan-drivers
Arch:           sudo pacman -S xorg-server-devel libxcursor libxrandr libxinerama libxi vulkan-swrast
Solus:          sudo eopkg it -c system.devel mesalib-devel libxrandr-devel libxcursor-devel libxi-devel libxinerama-devel vulkan
openSUSE:       sudo zypper install gcc libXcursor-devel libXrandr-devel Mesa-libGL-devel libXi-devel libXinerama-devel libXxf86vm-devel libvulkan1
Void:           sudo xbps-install -S base-devel xorg-server-devel libXrandr-devel libXcursor-devel libXinerama-devel vulkan-loader
Alpine:         sudo apk add gcc libxcursor-dev libxrandr-dev libxinerama-dev libxi-dev linux-headers mesa-dev vulkan-loader
NixOS:          nix-shell -p libGL pkg-config xorg.libX11.dev xorg.libXcursor xorg.libXi xorg.libXinerama xorg.libXrandr xorg.libXxf86vm mesa.drivers vulkan-loader
```

## Demo

You can run the Cogent Core Demo to verify Cogent Core is properly installed on your system:

```sh
go run cogentcore.org/core/examples/demo@main
```

You can also see the [web version of the demo](https://cogentcore.org/core/demo).

## Troubleshooting

If you run into any issues while installing Cogent Core or trying to run the demo, please re-read all of the install instructions above and make sure that you have followed them.

If the problem still happens, it is probably related to WebGPU and/or Vulkan. If you are on Linux, please follow these [instructions](https://linuxconfig.org/install-and-test-vulkan-on-linux) (or other ones for your distro) to make sure you have your device-specific Vulkan drivers installed.

If you still have problems, please file an [issue](https://github.com/cogentcore/core/issues). Please include the output of [webgpuinfo](https://github.com/cogentcore/core/tree/main/gpu/cmd/webgpuinfo) in your bug report. If you are on Linux, please also include the output of `vulkaninfo`.

After the demo is working, you can explore the [[basics]], [[tutorials]], and [[widgets]]. You can also see how to [[build]] apps.
