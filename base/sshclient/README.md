# SSH client

This package manages ssh connections to other hosts ("client" mode), using the Go standard ssh library.  It provides a similar interface as the [exec](../exec) package, e.g., with `Run`, `Output`, `Start` functions to execute commands, and its Config is based on `exec.Config`.

The `Client` type is used to establish a connection to a host, which persists until closed.

Each command execution process creates its own `ssh.Session` object that manages the process of running the command, within the scope of a persistently open Client connection.  This is much faster than opening a new connection for each command.  A map of open sessions is maintained in case of a `Start` command that runs the command asynchronously.

A good reference for similar package: https://github.com/helloyi/go-sshclient


