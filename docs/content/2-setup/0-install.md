Note: we recommend you read the [basics](../basics) and [tutorials](../tutorials) before you start developing with Cogent Core on your system. Complete all of the following steps to install Cogent Core:

1. Download and install Go from [the Go website](https://go.dev/doc/install) if you do not already have Go 1.22+ installed.
2. Run `go install cogentcore.org/core/cmd/core@main` to install the Cogent Core command line tool. You should run this command again each time you update to a new version of Cogent Core.
3. Run `core setup` to install platform-specific dependencies. Please first read all of the information below:
    * This installs the [xcode-tools](https://mac.install.guide/commandlinetools/4) on macOS, graphics libraries on Linux (listed below), and [w64devkit](https://github.com/skeeto/w64devkit) and [Git](https://git-scm.com/download/win) on Windows.
    * You should run the command from your home directory.
    * You may need to enter your password so that it can run `sudo` commands.
    * You should restart your shell/prompt/terminal after running the command.
    * On Windows, you must run the command from a shell/prompt/terminal running as administrator (but still in your home directory). You may need to follow steps in installers; the default options are fine. After running this command, you must run all future commands related to Cogent Core in Git Bash; please do **not** use Command Prompt or PowerShell.

<details>
<summary>

## Advanced
</summary>

The Linux install commands for each distro are listed below if you want to run them manually instead:

```sh
sudo apt-get install gcc libgl1-mesa-dev libegl1-mesa-dev mesa-vulkan-drivers xorg-dev # Debian / Ubuntu
sudo dnf install gcc libX11-devel libXcursor-devel libXrandr-devel libXinerama-devel mesa-libGL-devel libXi-devel libXxf86vm-devel # Fedora
```

</details>
