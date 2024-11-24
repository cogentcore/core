Note: we recommend you read the [[basics]] and [[tutorials]] before you start developing with Cogent Core on your system. Complete the following three steps to install Cogent Core:

1. Download and install Go from [the Go website](https://go.dev/doc/install) if you do not already have Go 1.22.6+ installed.

2. Run the following command to install the Cogent Core command line tool. You should run this command again each time you update to a new version of Cogent Core.

```sh
go install cogentcore.org/core/cmd/core@main
```

3. **After** first reading the notes for your platform below, run the following command to install platform-specific dependencies. You should restart your shell/prompt/terminal after running the command.

```sh
core setup
```

## macOS

This installs the [xcode-tools](https://mac.install.guide/commandlinetools/4). You may need to follow installer prompts. You **must** have Go 1.22.6+ installed due to an [XCode issue](https://github.com/golang/go/issues/68088).

## Windows

This installs [w64devkit](https://github.com/skeeto/w64devkit) and [Git](https://git-scm.com/download/win). You **must** run the setup command from a shell/prompt/terminal (such as Command Prompt or Git Bash) explicitly ["Run as administrator"](https://www.howtogeek.com/194041/how-to-open-the-command-prompt-as-administrator-in-windows-10/). You may need to follow steps in installers; the default options are fine. After running this command, you **must** run all future commands related to Cogent Core in Git Bash.

## Linux

This installs various graphics libraries on Linux as listed below. You may need to enter your password so that it can run `sudo` commands.

**TODO: List graphics libraries here...**

## Demo

You can run the Cogent Core Demo to verify Cogent Core is properly installed on your system:

```sh
go run cogentcore.org/core/examples/demo@main
```

If you run into any issues while trying to run the demo, please re-read all of the install instructions above and make sure that you have followed them. If the problem still happens, please [file a bug report](https://github.com/cogentcore/core/issues/new?assignees=&labels=bug&projects=&template=bug_report.yml).

You can also run the web version of the demo by going to [cogentcore.org/core/demo](https://cogentcore.org/core/demo).
