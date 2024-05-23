# Cogent Shell (cosh)

Cogent Shell (cosh) is a shell that combines the best parts of Go and command-based shell languages like `bash` to provide an integrated shell experience that allows you to easily run terminal commands while using Go for complicated logic. This allows you to write concise, elegant, and readable shell code that runs quickly on all platforms, and transpiles to Go (i.e, can be compiled by `go build`).

The simple idea is that each line is either Go or shell commands, determined in a fairly intuitive way mostly by the content at the start of the line (formal rules below), and they can be intermixed by wrapping Go within `{ }` and shell code from within backticks (`````).  We henceforth refer to shell code as `exec` code (in reference to the Go & Cogent `exec` package that we use to execute programs), given the potential ambituity of the entire `cosh` language being the shell. There are different syntactic formatting rules for these two domains of Go and Exec, within cosh:

* Go code is processed and formatted as usual (e.g., white space is irrelevant, etc).
* Exec code is space separated, like normal command-line invocations.

Examples:

```go
for i, f := range shell.SplitLines(`ls -la`) {   // `ls` executes returns string
    echo {i} {strings.ToLower(f)}           // {} surrounds go within shell
}
```

`splitLines` is a function that runs `strings.Split(arg, "\n")`, defined in the cosh standard library of such frequently-used helper functions.

# Special syntax

## Multiple statements per line

* Multiple statements can be combined on one line, separated by `;` as in regular Go and shell languages.  Critically, the language determination for the first statement determines the language for the remaining statements; you cannot intermix the two on one line, when using `;` 
# Exec mode

## Environment variables

* `set <var> <value>` (space delimited as in all exec mode, no equals)

## Output redirction

* Standard output redirect: `>` and `>&` (and `|`, `|&` if needed)

## Control flow

* Any error stops the script execution, except for statements wrapped in `[ ]`, indicating an "optional" statement, e.g.:

```sh
cd some; [mkdir sub]; cd sub
```

* `&` at the end of a statement runs in the background (as in bash) -- otherwise it waits until it completes before it continues.

* `jobs`, `fg`, `bg`, and `kill` builtin commands function as in usual bash.

## Exec functions (aliases)

Use the `command` keyword to define new functions for Exec mode execution, which can then be used like any other command, for example:

```sh
command list {
	ls -la args...
}
```

```sh
cd data
list *.tsv
```

The `command` is transpiled into a Go function that takes `args ...string`.  In the command function body, you can use the `args...` expression to pass all of the args, or `args[1]` etc to refer to specific positional indexes, as usual.

The command function name is registered so that the standard shell execution code can run the function, passing the args.  You can also call it directly from Go code using the standard parentheses expression.

# Makefile-like functionality

A few simple features of `cosh` enable powerful `make` functionality to be easily supported.  See [make.cosh](cmd/cosh/testdata/make.cosh) for an example, in `cmd/cosh/testdata/make.cosh`.

* Any line starting with a name that corresponds to a filename in the current directory with ".cosh" appended to it (e.g., `make` and `make.cosh`) is automatically recognized to run the `cosh` command on that file, with any args after the name appended as an `-e` argument to cosh.  Thus, the script line:

```sh
make build
```

is translated into `cosh make.cosh -e "build"` and run.  The resulting code that is run is just the contents of the `make.cosh` file with the additional contents of the `-e` expression appended at the end, such that any `command`s that are defined within `make.cosh` would be run by the `-e` expression.

Note that there is nothing special about the name `make` here, so this can be done with any file.

The `make` package defines a number of useful utility functions that accomplish the standard dependency and file timestamp checking functionality from the standard `make` command, as in the [magefile](https://magefile.org/dependencies/) system.  Note that the cosh direct exec command syntax makes the resulting make files much closer to a standard bash-like Makefile, while still having all the benefits of Go control and expressions, compared to magefile.

TODO: implement and document.

**NOTE:** for now, if it doesn't find a new file in the current directory, type `cd .` and it will refresh the cache of files in current directory.  TODO: fix this with time-stamp of some sort!

# SSH connections to remote hosts

Any number of active SSH connections can be maintained and used dynamically within a script, including simple ways of copying data among the different hosts (including the local host).  The Go mode execution is always on the local host in one running process, and only the shell commands are executed remotely, enabling a unique ability to easily coordinate and distribute processing and data across various hosts.

Each host maintains its own working directory and environment variables, which can be configured and re-used by default whenever using a given host.

* `cossh hostname.org [name]`  establishes a connection, using given optional name to refer to this connection.  If the name is not provided, a sequential number will be used, starting with 1, with 0 referring always to the local host.

* `@name` then refers to the given host in all subsequent commands, with `@0` referring to the local host where the cosh script is running.

### Explicit per-command specification of host

```sh
@name cd subdir; ls
```

### Default host

```sh
@name // or:
cossh @name
```

uses the given host for all subsequent commands (unless explicitly specified), until the default is changed.  Use `cossh @0` to return to localhost.

### Redirect input / output among hosts

The output of a remote host command can be sent to a file on the local host:
```sh
@name cat hostfile.tsv > @0:localfile.tsv
```
Note the use of the `:` colon delimiter after the host name here.  TODO: You cannot send output to a remote host file (e.g., `> @host:remotefile.tsv`) -- maybe with sftp?

The output of any command can also be piped to a remote host as its standard input:
```sh
ls *.tsv | @host cat > files.txt
```

### scp to copy files easily

The builtin `scp` function allows easy copying of files across hosts, using the persistent connections established with `cossh` instead of creating new connections as in the standard scp command.

`scp` is _always_ run from the local host, with the remote host filename specified as `@name:remotefile`

```sh
scp @name:hostfile.tsv localfile.tsv
```

TODO: Importantly, file wildcard globbing works as expected:
```sh
scp @name:*.tsv @0:data/
```

and entire directories can be copied, as in `cp -a` or `cp -r` (this behavior is automatic and does not require  a flag).

### Close connections

```sh
cossh close
```

Will close all active connections and return the default host to @0.  All active connections are also automatically closed when the shell terminates.

# Other Utilties

** need a replacement for findnm -- very powerful but garbage..

# Rules for Go vs. Shell determination

The critical extension from standard Go syntax is for lines that are processed by the `Exec` functions, used for running arbitrary programs on the user's executable path.  Here are the rules (word = IDENT token):

* Backticks "``" anywhere:  Exec.  Returns a `string`.
* Within Exec, `{}`: Go
* Line starts with `Go` Keyword: Go
* Line is one word: Exec
* Line starts with `path`: Exec
* Line starts with `"string"`: Exec
* Line starts with `word word`: Exec
* Line starts with `word {`: Exec
* Otherwise: Go

# TODO:

* likewise, need to run everything effectively as a bg job with our own explicit Wait, which we can then communicate with to move from fg to bg.


