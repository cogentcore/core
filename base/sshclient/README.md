# SSH client

This package manages ssh connections to other hosts ("client" mode), using the Go standard ssh library.  It provides a similar interface as the [exec](../exec) package, e.g., with `Run`, `Output`, `Start` functions to execute commands, and its Config is based on `exec.Config`.

The `Client` type is used to establish a connection to a host, which persists until closed.

Each command execution process creates its own `ssh.Session` object that manages the process of running the command, within the scope of a persistently open Client connection.  This is much faster than opening a new connection for each command.  A map of open sessions is maintained in case of a `Start` command that runs the command asynchronously.

# Environment and interactive commands

Each command runs in a new shell on the host that is not interactive, for which a standard `~/.bashrc` returns before it sets the PATH and other variables of the user. So when it connects, the client gets the environment of an interactive login shell of the user once, and each command starts by setting the variables that differ (`Config.LoginEnv`, on by default).

A command run with its input from a terminal gets that input while it runs, in a terminal on the host with the same size, so that interactive programs and Control+C work, and `Client.Shell` runs an interactive shell on the host. With `RunContext` etc, a command is interrupted when its context is done.

# Lost connections

A connection can be lost without being closed, as when a laptop sleeps or changes networks, and then everything waiting on it would wait forever. So the client sends a keepalive request to the host every `Config.KeepAlive` (15 seconds by default), and closes the connection if the host does not reply within `Config.Timeout` (also 15 seconds, which also limits connecting), so that the commands waiting on it fail. The next command, or `Client.CheckConnection`, connects again to the same host, keeping the current directory and environment, as when the network comes back. While the host cannot be reached, it tries again only after the `Timeout`, so that commands fail fast.

A good reference for similar package: https://github.com/helloyi/go-sshclient

# Copying files

Files are copied to and from hosts with the SFTP protocol of https://github.com/pkg/sftp, on the connection of the client: `Copy` copies files between any hosts, including the local one, as `cp -r` does, with wildcards, and `Client.OpenFile` opens a file on the host.

The `sshtest` package provides an SSH server for tests, which runs commands and serves SFTP in the test process, and can simulate the loss of the network (`Server.NetworkDown` and `NetworkUp`).


