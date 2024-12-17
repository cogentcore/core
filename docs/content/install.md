You can complete the three steps below to **install** Cogent Core on your system. You can also explore the [[#demo|demo]], [[basics]], [[tutorials]], and [[widgets]].

1. Download and install Go from [the Go website](https://go.dev/doc/install) if you do not already have Go 1.22.6+ installed.

2. Run the following command to install the Cogent Core command line tool. You should run this command again each time you update to a new version of Cogent Core.

```sh
go install cogentcore.org/core/cmd/core@main
```

3. **After** reading the info for your platform below, run the following command to install platform-specific dependencies. You **must** restart your shell/prompt/terminal after running the command.

```sh
core setup
```

## macOS

This installs the [xcode-tools](https://mac.install.guide/commandlinetools/4). You may need to follow installer prompts. You **must** have Go 1.22.6+ installed due to an [Xcode issue](https://github.com/golang/go/issues/68088).

## Windows

This installs [w64devkit](https://github.com/skeeto/w64devkit) and [Git](https://git-scm.com/download/win). You **must** run the setup command from a shell/prompt/terminal (such as Command Prompt or Git Bash) explicitly ["Run as administrator"](https://www.howtogeek.com/194041/how-to-open-the-command-prompt-as-administrator-in-windows-10/). You may need to follow steps in installers; the default options are good. After running this command, you **must** run all future commands related to Cogent Core in Git Bash.

## Linux

This installs various graphics libraries. You may need to enter your password so that it can run `sudo` commands. On some devices and distros, you may also need to install a custom Lavapipe Vulkan driver. For example, on Arch Linux without a dedicated GPU, you need to install [vulkan-swrast](https://archlinux.org/packages/extra/x86_64/vulkan-swrast/).

If you want to install manually instead, the commands for each distro are listed below.

<!-- To update this, copy the output of [cogentcore.org/core/cmd/core/cmd.TestLinuxDistroString]; DO NOT EDIT manually -->

```sh
Debian/Ubuntu:  sudo apt install golang gcc libgl1-mesa-dev libegl1-mesa-dev mesa-vulkan-drivers xorg-dev
Fedora:         sudo dnf install golang golang-misc gcc libX11-devel libXcursor-devel libXrandr-devel libXinerama-devel mesa-libGL-devel libXi-devel libXxf86vm-devel
Arch:           sudo pacman -S go xorg-server-devel libxcursor libxrandr libxinerama libxi vulkan-swrast
Solus:          sudo eopkg it -c system.devel golang mesalib-devel libxrandr-devel libxcursor-devel libxi-devel libxinerama-devel
openSUSE:       sudo zypper install go gcc libXcursor-devel libXrandr-devel Mesa-libGL-devel libXi-devel libXinerama-devel libXxf86vm-devel
Void:           sudo xbps-install -S go base-devel xorg-server-devel libXrandr-devel libXcursor-devel libXinerama-devel
Alpine:         sudo apk add go gcc libxcursor-dev libxrandr-dev libxinerama-dev libxi-dev linux-headers mesa-dev
NixOS:          nix-shell -p libGL pkg-config xorg.libX11.dev xorg.libXcursor xorg.libXi xorg.libXinerama xorg.libXrandr xorg.libXxf86vm
```

## Demo

You can run the Cogent Core Demo to verify Cogent Core is properly installed on your system:

```sh
go run cogentcore.org/core/examples/demo@main
```

If you run into any issues while trying to run the demo, please re-read all of the install instructions above and make sure that you have followed them. If the problem still happens, please [file a bug report](https://github.com/cogentcore/core/issues/new?assignees=&labels=bug&projects=&template=bug_report.yml).

You can also run the [web version of the demo](https://cogentcore.org/core/demo).

After you run the demo, you can explore the [[basics]], [[tutorials]], and [[widgets]].
