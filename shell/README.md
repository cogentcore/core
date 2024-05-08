# Cogent Shell (cosh)

Cogent Shell (cosh) is a shell that combines the best parts of Go and bash to provide an integrated shell experience that allows you to easily run terminal commands while using Go for complicated logic. This allows you to write concise, elegant, and readable shell code that runs quickly on all platforms and transpiles to Go.

Examples:

```go
for i, f := range `ls` {   // `ls` executes shell and ranges over result
    echo {i} {strings.ToLower(f)}           // {} surrounds go within shell
    git status {f}
}
```

The simple idea is that each line is either Go or shell commands, determined in a fairly intuitive way mostly on the content at the start of the line (formal rules below), and they can be intermixed by wrapping Go within `{ }` and shell from within backticks. 

* Go code is processed and formatted as usual (e.g., white space is irrelevant, etc).
* Shell code is space separated, like normal command-line invocations.

# Special syntax

* Multiple statements can be combined on one line, separated by `;` as in regular Go and shell languages.  Critically, the language determination for the first statement determines the language for the remaining statements; you cannot intermix the two.

# Shell mode

## Environment variables

* `set <var> <value>` (space delimited as in all shell)

## Output redirction

* Standard output redirect: `>` and `>&` (and `|`, `|&` if needed)

## Control flow

* Any error stops the script execution, except for statements wrapped in `[ ]`, indicating an "optional" statement, e.g.:

```sh
cd some; [mkdir sub]; cd sub
```

* `&` at the end of a statement runs in the background (as in bash) -- otherwise it waits until it completes before it continues.

* `jobs`, `fg`, `bg`, and `kill` builtin commands function as in usual bash.

# SSH connections to remote hosts

Any number of active SSH connections can be maintained and used dynamically within a script, including simple ways of copying data among the different hosts (including the local host).  The Go level execution is always on the local host in one running process, and only the shell commands are executed remotely, enabling a unique ability to easily coordinate and distribute processing and data across various hosts.

Each host maintains its own working directory and environment variables, which can be configured and re-used by default whenever using a given host.

* `cossh hostname.org [name]`  establishes a connection, using given optional name to refer to this connection.  If the name is not provided, a sequential number will be used, starting with 1, with 0 referring always to the local host.

* `@name` refers to the given host

### Explicit per-command determination of where to run a command:

```sh
@name cd subdir; ls
```

Note that @0 always refers to the localhost.

### Set default command host for subsequent shell commands.

```sh
cossh @name
```

use `cossh @0` to return to localhost.

### Redirect input / output among hosts

```sh
cat @0:localfile.tsv > @host:remotefile.tsv
```

note the use of colon after host name identifier when specifying files.  The files in each host are always relative to the current working directory for that host.

All file redirect logic applies, including pipes between commands across hosts, e.g.,:

```sh
@0 ls *.tsv | @name git add
```

### Close connections

```sh
cossh close
```

Will close all active connections and return the default host to @0.  All active connections are also automatically closed when the shell terminates.

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



