# Goal: Go augmented language

Goal is an augmented version of the Go language, which combines the best parts of Go, `bash`, and Python, to provide and integrated shell and numerical expression processing experience, which can be combined with the [yaegi](https://github.com/traefik/yaegi) interpreter to provide an interactive "REPL" (read, evaluate, print loop).

Goal transpiles directly into Go, so it automatically leverages all the great features of Go, and remains fully compatible with it.  The augmentation is designed to overcome some of the limitations of Go in specific domains:

* Shell scripting, where you want to be able to directly call other executable programs with arguments, without having to navigate all the complexity of the standard [os.exec](https://pkg.go.dev/os/exec) package.

* Numerical / math / data processing, where you want to be able to write simple mathematical expressions operating on vectors, matricies and other more powerful data types, without having to constantly worry about type conversions and need extended indexing and slicing expressions. Python is the dominant language here precisely because it lets you ignore type information and write such expressions.

The main goal of Goal is to achieve a "best of both worlds" solution that retains all the type safety and explicitness of Go for all the surrounding control flow and large-scale application logic, while also allowing for a more relaxed syntax in specific, well-defined domains where the Go language has been a barrier.  Thus, unlike Python where there are various weak attempts to try to encourage better coding habits, Goal retains in its Go foundation a fundamentally scalable, "industrial strength" language that has already proven its worth in countless real-world applications.

For the shell scripting aspect of Goal, the simple idea is that each line of code is either Go or shell commands, determined in a fairly intuitive way mostly by the content at the start of the line (formal rules below). If a line starts off with something like `ls -la...` then it is clear that it is not valid Go code, and it is therefore processed as a shell command.

You can intermix Go within a shell line by wrapping an expression with `{ }` braces, and a Go expression can contain shell code by using `$`.  Here's an example:
```go
for i, f := range goalib.SplitLines($ls -la$) {  // ls executes, returns string
    echo {i} {strings.ToLower(f)}              // {} surrounds Go within shell
}
```
where `goalib.SplitLines` is a function that runs `strings.Split(arg, "\n")`, defined in the `goalib` standard library of such frequently-used helper functions.

For cases where most of the code is standard Go with relatively infrequent use of shell expressions, or in the rare cases where the default interpretation doesn't work, you can explicitly tag a line as shell code using `$`:

```go
$ chmod +x *.goal
```

For mathematical expressions, we use `#` symbols (`#` = number) to demarcate such expressions. Often you will write entire lines of such expressions:
```go
# x := 1. / (1. + exp(-wts[:, :, :n] * acts[:]))
```
You can also intermix within Go code:
```go
for _, x := range #[1,2,3]# {
    fmt.Println(#x^2#)
}
```

In general, the math mode syntax in Goal is designed to be as compatible with Python NumPy / scipy syntax as possible, while also adding a few Go-specific additions as well -- see [Math mode](#math-mode) for details.  All elements of a Goal math expression are [tensors](../tensor), which can represent everything from a scalar to an n-dimenstional tensor.  These are called an "ndarray" in NumPy terms.

The rationale and mnemonics for using `$` and `#` are as follows:

* These are two of the three symbols that are not part of standard Go syntax (`@` being the other).

* `$` can be thought of as "S" in _S_hell, and is often used for a `bash` prompt, and many bash examples use it as a prefix. Furthermore, in bash, `$( )` is used to wrap shell expressions.

* `#` is commonly used to refer to numbers. It is also often used as a comment syntax, but on balance the number semantics and uniqueness relative to Go syntax outweigh that issue.

# Examples

Here are a few useful examples of Goal code:

You can easily perform handy duration and data size formatting:

```go
22010706 * time.Nanosecond  // 22.010706ms
datasize.Size(44610930)     // 42.5 MB
```

# Shell mode

## Environment variables

* `set <var> <value>` (space delimited as in all shell mode, no equals)

## Output redirction

* Standard output redirect: `>` and `>&` (and `|`, `|&` if needed)

## Control flow

* Any error stops the script execution, except for statements wrapped in `[ ]`, indicating an "optional" statement, e.g.:

```sh
cd some; [mkdir sub]; cd sub
```

* `&` at the end of a statement runs in the background (as in bash) -- otherwise it waits until it completes before it continues.

* `jobs`, `fg`, `bg`, and `kill` builtin commands function as in usual bash.

## Shell functions (aliases)

Use the `command` keyword to define new functions for Shell mode execution, which can then be used like any other command, for example:

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

## Script Files and Makefile-like functionality

As with most scripting languages, a file of goal code can be made directly executable by appending a "shebang" expression at the start of the file:

```sh
#!/usr/bin/env goal
```

When executed this way, any additional args are available via an `args []any` variable, which can be passed to a command as follows:
```go
install {args...}
```
or by referring to specific arg indexes etc.

To make a script behave like a standard Makefile, you can define different `command`s for each of the make commands, and then add the following at the end of the file to use the args to run commands:

```go
goal.RunCommands(args)
```

See [make](cmd/goal/testdata/make) for an example, in `cmd/goal/testdata/make`, which can be run for example using:

```sh
./make build
```

Note that there is nothing special about the name `make` here, so this can be done with any file.

The `make` package defines a number of useful utility functions that accomplish the standard dependency and file timestamp checking functionality from the standard `make` command, as in the [magefile](https://magefile.org/dependencies/) system.  Note that the goal direct shell command syntax makes the resulting make files much closer to a standard bash-like Makefile, while still having all the benefits of Go control and expressions, compared to magefile.

TODO: implement and document above.

## SSH connections to remote hosts

Any number of active SSH connections can be maintained and used dynamically within a script, including simple ways of copying data among the different hosts (including the local host).  The Go mode execution is always on the local host in one running process, and only the shell commands are executed remotely, enabling a unique ability to easily coordinate and distribute processing and data across various hosts.

Each host maintains its own working directory and environment variables, which can be configured and re-used by default whenever using a given host.

* `cossh hostname.org [name]`  establishes a connection, using given optional name to refer to this connection.  If the name is not provided, a sequential number will be used, starting with 1, with 0 referring always to the local host.

* `@name` then refers to the given host in all subsequent commands, with `@0` referring to the local host where the goal script is running.

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

Importantly, file wildcard globbing works as expected:
```sh
scp @name:*.tsv @0:data/
```

and entire directories can be copied, as in `cp -a` or `cp -r` (this behavior is automatic and does not require  a flag).

### Close connections

```sh
cossh close
```

Will close all active connections and return the default host to @0.  All active connections are also automatically closed when the shell terminates.

## Other Utilties

** TODO: need a replacement for findnm -- very powerful but garbage..

## Rules for Go vs. Shell determination

These are the rules used to determine whether a line is Go vs. Shell (word = IDENT token):

* `$` at the start: Shell.
* Within Shell, `{}`: Go
* Within Go, `$ $`: Shell
* Line starts with `go` keyword: if no `( )` then Shell, else Go
* Line is one word: Shell
* Line starts with `path` expression (e.g., `./myexec`) : Shell
* Line starts with `"string"`: Shell
* Line starts with `word word`: Shell
* Line starts with `word {`: Shell
* Otherwise: Go

TODO: update above

## Multiple statements per line

* Multiple statements can be combined on one line, separated by `;` as in regular Go and shell languages.  Critically, the language determination for the first statement determines the language for the remaining statements; you cannot intermix the two on one line, when using `;` 

# Math mode

The math mode in Goal is designed to be generally compatible with Python NumPy / SciPy syntax, so that the widespread experience with that syntax transfers well to Goal. This syntax is also largely compatible with MATLAB and other languages as well. However, we did not fully replicate the NumPy syntax, instead choosing to clean up a few things and generally increase consistency with Go.

In general the Goal global functions are named the same as NumPy, without the `np.` prefix, which improves readability. It should be very straightforward to write a conversion utility that converts existing NumPy code into Goal code, and that is a better process than trying to make Goal itself perfectly compatible.

All elements of a Goal math expression are [tensors](../tensor) (i.e., `tensor.Tensor`), which can represent everything from a scalar to an n-dimenstional tensor, with different _views_ that support the arbitrary slicing and flexible forms of indexing documented in the table below.  These are called an `ndarray` in NumPy terms.  See [array vs. tensor](https://numpy.org/doc/stable/user/numpy-for-matlab-users.html#array-or-matrix-which-should-i-use) NumPy docs for more information.  Note that Goal does not have a distinct `matrix` type; everything is a tensor, and when these are 2D, they function appropriately via the [matrix](../tensor/matrix) package.

The _view_ versions of `Tensor` include `Sliced`, `Reshaped`,  `Masked`, `Indexed`, and `Rows`, each of which wraps around another "source" `Tensor`, and provides its own way of accessing the underlying data:

* `Sliced` has an arbitrary set of indexes for each dimension, so access to values along that dimension go through the indexes.  Thus, you could reverse the order of the columns (dimension 1), or only operate on a subset of them.

* `Masked` has a `tensor.Bool` tensor that filters access to the underlying source tensor through a mask: anywhere the bool value is `false`, the corresponding source value is not settable, and returns `NaN` (missing value) when accessed.

* `Indexed` uses a tensor of indexes where the final, innermost dimension is the same size as the number of dimensions in the wrapped source tensor. The overall shape of this view is that of the remaining outer dimensions of the Indexes tensor, and like other views, assignment and return values are taken from the corresponding indexed value in the wrapped source tensor.

    The current NumPy version of indexed is rather complex and difficult for many people to understand, as articulated in this [NEP 21 proposal](https://numpy.org/neps/nep-0021-advanced-indexing.html). The `Indexed` view at least provides a simpler way of representing the indexes into the source tensor, instead of requiring multiple parallel 1D arrays.

* `Rows` is an optimized version of `Sliced` with indexes only for the first, outermost, _row_ dimension.

The following sections provide a full list of equivalents between the `tensor` Go code, Goal, NumPy, and MATLAB, based on the table in [numpy-for-matlab-users](https://numpy.org/doc/stable/user/numpy-for-matlab-users.html).
* The _same:_ in Goal means that the same NumPy syntax works in Goal, minus the `np.` prefix, and likewise for _or:_ (where Goal also has additional syntax).
* In the `tensor.Go` code, we sometimes just write a scalar number for simplicity, but these are actually `tensor.NewFloat64Scalar` etc.
* Goal also has support for `string` tensors, e.g., for labels, and operators such as addition that make sense for strings are supported. Otherwise, strings are automatically converted to numbers using the `tensor.Float` interface. If you have any doubt about whether you've got a `tensor.Float64` when you expect one, use `tensor.AsFloat64Tensor` which makes sure.

## Tensor shape

| `tensor` Go  |   Goal      | NumPy  | MATLAB | Notes            |
| ------------ | ----------- | ------ | ------ | ---------------- |
| `a.NumDim()` | `ndim(a)` or `a.ndim` | `np.ndim(a)` or `a.ndim`   | `ndims(a)` | number of dimensions of tensor `a` |
| `a.Len()`    | `len(a)` or `a.len` or: | `np.size(a)` or `a.size`   | `numel(a)` | number of elements of tensor `a` |
| `a.Shape().Sizes` | same: | `np.shape(a)` or `a.shape` | `size(a)`  | "size" of each dimension in a; `shape` returns a 1D `int` tensor |
| `a.Shape().Sizes[1]` | same: | `a.shape[1]` | `size(a,2)` | the number of elements of the 2nd dimension of tensor `a` |
| `tensor.Reshape(a, 10, 2)` | same except no `a.shape = (10,2)`: | `a.reshape(10, 2)` or `np.reshape(a, 10, 2)` or `a.shape = (10,2)` | `reshape(a,10,2)` | set the shape of `a` to a new shape that has the same total number of values (len or size); No option to change order in Goal: always row major; Goal does _not_ support direct shape assignment version. |
| `tensor.Reshape(a, tensor.AsIntSlice(sh)...)` | same: | `a.reshape(10, sh)` or `np.reshape(a, sh)` |  `reshape(a,sh)` | set shape based on list of dimension sizes in tensor `sh` |
| `tensor.Reshape(a, -1)` or `tensor.As1D(a)` | same: | `a.reshape(-1)` or `np.reshape(a, -1)` | `reshape(a,-1)` | a 1D vector view of `a`; Goal does not support `ravel`, which is nearly identical. |
| `tensor.Flatten(a)` | same: | `b = a.flatten()`   | `b=a(:)` | returns a 1D copy of a |
| `b := tensor.Clone(a)` | `b := copy(a)` or: | `b = a.copy()` | `b=a`  | direct assignment `b = a` in Goal or NumPy just makes variable b point to tensor a; `copy` is needed to generate new underlying values (MATLAB always makes a copy) |
| `tensor.Squeeze(a)` | same: |`a.squeeze()` | `squeeze(a)` | remove singleton dimensions of tensor `a`. |


## Constructing

| `tensor` Go  |   Goal      | NumPy  | MATLAB | Notes            |
| ------------ | ----------- | ------ | ------ | ---------------- |
| `tensor.NewFloat64FromValues(` `[]float64{1, 2, 3})` | `[1., 2., 3.]` | `np.array([1., 2., 3.])` | `[ 1 2 3 ]` | define a 1D tensor |
|  | `[[1., 2., 3.], [4., 5., 6.]]` or: | `(np.array([[1., 2., 3.], [4., 5., 6.]])` | `[ 1 2 3; 4 5 6 ]` | define a 2x3 2D tensor |
|  |  | `[[a, b], [c, d]]` or `block([[a, b], [c, d]])` | `np.block([[a, b], [c, d]])` | `[ a b; c d ]` | construct a matrix from blocks `a`, `b`, `c`, and `d` |
| `tensor.NewFloat64(3,4)` | `zeros(3,4)` | `np.zeros((3, 4))` | `zeros(3,4)` | 3x4 2D tensor of float64 zeros; Goal does not use "tuple" so no double parens |
| `tensor.NewFloat64(3,4,5)` | `zeros(3, 4, 5)` | `np.zeros((3, 4, 5))` | `zeros(3,4,5)` | 3x4x5 three-dimensional tensor of float64 zeros |
| `tensor.NewFloat64Ones(3,4)` | `ones(3, 4)`  | `np.ones((3, 4))` | `ones(3,4)` | 3x4 2D tensor of 64-bit floating point ones |
| `tensor.NewFloat64Full(5.5, 3,4)` | `full(5.5, 3, 4)` | `np.full((3, 4), 5.5)` | ? | 3x4 2D tensor of 5.5; Goal variadic arg structure requires value to come first |
| `tensor.NewFloat64Rand(3,4)` | `rand(3, 4)` or `slrand(c, fi, 3, 4)` | `rng.random(3, 4)` | `rand(3,4)` | 3x4 2D float64 tensor with uniform random 0..1 elements; `rand` uses current Go `rand` source, while `slrand` uses [gosl](../gpu/gosl/slrand) GPU-safe call with counter `c` and function index `fi` and key = index of element |
| TODO: |  |`np.concatenate((a,b),1)` or `np.hstack((a,b))` or `np.column_stack((a,b))` or `np.c_[a,b]` | `[a b]` | concatenate columns of a and b |
| TODO: |  |`np.concatenate((a,b))` or `np.vstack((a,b))` or `np.r_[a,b]` | `[a; b]` | concatenate rows of a and b |
| TODO: |  |`np.tile(a, (m, n))`    | `repmat(a, m, n)` | create m by n copies of a |
| TODO: |  |`a[np.r_[:len(a),0]]`  | `a([1:end 1],:)`  | `a` with copy of the first row appended to the end |

## Ranges and grids

See [NumPy](https://numpy.org/doc/stable/user/how-to-partition.html) docs for details.

| `tensor` Go  |   Goal      | NumPy  | MATLAB | Notes            |
| ------------ | ----------- | ------ | ------ | ---------------- |
| `tensor.NewIntRange(1, 11)` | same: |`np.arange(1., 11.)` or `np.r_[1.:11.]` or `np.r_[1:10:10j]` | `1:10` | create an increasing vector; `arange` in goal is always ints; use `linspace` or `tensor.AsFloat64` for floats |
|  | same: |`np.arange(10.)` or `np.r_[:10.]` or `np.r_[:9:10j]` | `0:9` | create an increasing vector; 1 arg is the stop value in a slice |
|  |  |`np.arange(1.,11.)` `[:, np.newaxis]` | `[1:10]'` | create a column vector |
| `t.NewFloat64` `SpacedLinear(` `1, 3, 4, true)` | `linspace(1,3,4,true)` |`np.linspace(1,3,4)` | `linspace(1,3,4)` | 4 equally spaced samples between 1 and 3, inclusive of end (use `false` at end for exclusive) |
|  |  |`np.mgrid[0:9.,0:6.]` or `np.meshgrid(r_[0:9.],` `r_[0:6.])` | `[x,y]=meshgrid(0:8,0:5)` | two 2D tensors: one of x values, the other of y values |
|  |  |`ogrid[0:9.,0:6.]` or `np.ix_(np.r_[0:9.],` `np.r_[0:6.]` | | the best way to eval functions on a grid |
|  |  |`np.meshgrid([1,2,4],` `[2,4,5])` | `[x,y]=meshgrid([1,2,4],[2,4,5])` |  |
|  |  |`np.ix_([1,2,4],` `[2,4,5])`    |  | the best way to eval functions on a grid |

## Basic indexing

See [NumPy basic indexing](https://numpy.org/doc/stable/user/basics.indexing.html#basic-indexing). Tensor Go uses the `Reslice` function for all cases (repeated `tensor.` prefix replaced with `t.` to take less space). Here you can clearly see the advantage of Goal in allowing significantly more succinct expressions to be written for accomplishing critical tensor functionality.

| `tensor` Go  |   Goal      | NumPy  | MATLAB | Notes            |
| ------------ | ----------- | ------ | ------ | ---------------- |
| `t.Reslice(a, 1, 4)` | same: |`a[1, 4]` | `a(2,5)` | access element in second row, fifth column in 2D tensor `a` |
| `t.Reslice(a, -1)` | same: |`a[-1]` | `a(end)` | access last element |
| `t.Reslice(a,` `1, t.FullAxis)` | same: |`a[1]` or `a[1, :]` | `a(2,:)` | entire second row of 2D tensor `a`; unspecified dimensions are equivalent to `:` (could omit second arg in Reslice too) |
| `t.Reslice(a,` `Slice{Stop:5})` | same: |`a[0:5]` or `a[:5]` or `a[0:5, :]` | `a(1:5,:)` | 0..4 rows of `a`; uses same Go slice ranging here: (start:stop) where stop is _exclusive_ |
| `t.Reslice(a,` `Slice{Start:-5})` | same: |`a[-5:]` | `a(end-4:end,:)` | last 5 rows of 2D tensor `a` |
| `t.Reslice(a,` `t.NewAxis,` `Slice{Start:-5})` | same: |`a[newaxis, -5:]` | ? | last 5 rows of 2D tensor `a`, as a column vector |
| `t.Reslice(a,` `Slice{Stop:3},` `Slice{Start:4, Stop:9})` | same: |`a[0:3, 4:9]` | `a(1:3,5:9)` | The first through third rows and fifth through ninth columns of a 2D tensor, `a`. |
| `t.Reslice(a,` `Slice{Start:2,` `Stop:25,` `Step:2}, t.FullAxis)` | same: |`a[2:21:2,:]` | `a(3:2:21,:)` | every other row of `a`, starting with the third and going to the twenty-first |
| `t.Reslice(a,` `Slice{Step:2},` `t.FullAxis)` | same: |`a[::2, :]`  | `a(1:2:end,:)` | every other row of `a`, starting with the first |
| `t.Reslice(a,`, `Slice{Step:-1},` `t.FullAxis)` | same: |`a[::-1,:]`  | `a(end:-1:1,:) or flipud(a)` | `a` with rows in reverse order |
| `t.Clone(t.Reslice(a,` `1, t.FullAxis))` | `b = copy(a[1, :])` or: | `b = a[1, :].copy()` | `y=x(2,:)` | without the copy, `y` would point to a view of values in `x`; `copy` creates distinct values, in this case of _only_ the 2nd row of `x` -- i.e., it "concretizes" a given view into a literal, memory-continuous set of values for that view. |
| `tmath.Assign(` `t.Reslice(a,` `Slice{Stop:5}),` `t.NewIntScalar(2))` | same: |`a[:5] = 2` | `a(1:5,:) = 2` | assign the value 2 to 0..4 rows of `a` |
| (you get the idea) | same: |`a[:5] = b[:, :5]` | `a(1:5,:) = b(:, 1:5)` | assign the values in the first 5 columns of `b` to the first 5 rows of `a` |

## Boolean tensors and indexing

See [NumPy boolean indexing](https://numpy.org/doc/stable/user/basics.indexing.html#boolean-array-indexing).

Note that Goal only supports boolean logical operators (`&&` and `||`) on boolean tensors, not the single bitwise operators `&` and `|`.

| `tensor` Go  |   Goal      | NumPy  | MATLAB | Notes            |
| ------------ | ----------- | ------ | ------ | ---------------- |
| `tmath.Greater(a, 0.5)` | same: | `(a > 0.5)` | `(a > 0.5)` | `bool` tensor of shape `a` with elements `(v > 0.5)` |
| `tmath.And(a, b)` | `a && b` | `logical_and(a,b)` | `a & b` | element-wise AND operator on `bool` tensors |
| `tmath.Or(a, b)` | `a \|\| b` | `np.logical_or(a,b)` | `a \| b` | element-wise OR operator on `bool` tensors | 
| `tmath.Negate(a)` | `!a` | ? | ? | element-wise negation on `bool` tensors | 
| `tmath.Assign(` `tensor.Mask(a,` `tmath.Less(a, 0.5),` `0)` | same: |`a[a < 0.5]=0` | `a(a<0.5)=0` | `a` with elements less than 0.5 zeroed out |
| `tensor.Flatten(` `tensor.Mask(a,` `tmath.Less(a, 0.5)))` | same: |`a[a < 0.5].flatten()` | ? | a 1D list of the elements of `a` < 0.5 (as a copy, not a view) |
| `tensor.Mul(a,` `tmath.Greater(a, 0.5))` | same: |`a * (a > 0.5)` | `a .* (a>0.5)` | `a` with elements less than 0.5 zeroed out |

## Advanced index-based indexing

See [NumPy integer indexing](https://numpy.org/doc/stable/user/basics.indexing.html#integer-array-indexing).  Note that the current NumPy version of indexed is rather complex and difficult for many people to understand, as articulated in this [NEP 21 proposal](https://numpy.org/neps/nep-0021-advanced-indexing.html). 

**TODO:** not yet implemented:

| `tensor` Go  |   Goal      | NumPy  | MATLAB | Notes            |
| ------------ | ----------- | ------ | ------ | ---------------- |
|  |  |`a[np.ix_([1, 3, 4], [0, 2])]` | `a([2,4,5],[1,3])` | rows 2,4 and 5 and columns 1 and 3. |
|  |  |`np.nonzero(a > 0.5)` | `find(a > 0.5)` | find the indices where (a > 0.5) |
|  |  |`a[:, v.T > 0.5]` | `a(:,find(v>0.5))` | extract the columns of `a` where column vector `v` > 0.5 |
|  |  |`a[:,np.nonzero(v > 0.5)[0]]` | `a(:,find(v > 0.5))` | extract the columns of `a` where vector `v` > 0.5 |
|  |  |`a[:] = 3` | `a(:) = 3` | set all values to the same scalar value |
|  |  |`np.sort(a)` or `a.sort(axis=0)` | `sort(a)` | sort each column of a 2D tensor, `a` |
|  |  |`np.sort(a, axis=1)` or `a.sort(axis=1)` | `sort(a, 2)` | sort the each row of 2D tensor, `a` |
|  |  |`I = np.argsort(a[:, 0]); b = a[I,:]` | `[b,I]=sortrows(a,1)`  | save the tensor `a` as tensor `b` with rows sorted by the first column |
|  |  |`np.unique(a)` | `unique(a)` | a vector of unique values in tensor `a` |

## Basic math operations (add, multiply, etc)

In Goal and NumPy, the standard `+, -, *, /` operators perform _element-wise_ operations because those are well-defined for all dimensionalities and are consistent across the different operators, whereas matrix multiplication is specifically used in a 2D linear algebra context, and is not well defined for the other operators.

| `tensor` Go  |   Goal      | NumPy  | MATLAB | Notes            |
| ------------ | ----------- | ------ | ------ | ---------------- |
| `tmath.Add(a,b)` | same: |`a + b` | `a .+ b` | element-wise addition; Goal does this string-wise for string tensors |
| `tmath.Mul(a,b)` | same: |`a * b` | `a .* b` | element-wise multiply |
| `tmath.Div(a,b)` | same: |`a/b`   | `a./b` | element-wise divide. _important:_ this always produces a floating point result. |
| `tmath.Mod(a,b)` | same: |`a%b`   | `a./b` | element-wise modulous (works for float and int) |
| `tmath.Pow(a,3)` | same: | `a**3`  | `a.^3` | element-wise exponentiation |
| `tmath.Cos(a)`   | same: | `cos(a)` | `cos(a)` | element-wise function application |

## 2D Matrix Linear Algebra

| `tensor` Go  |   Goal      | NumPy  | MATLAB | Notes            |
| ------------ | ----------- | ------ | ------ | ---------------- |
| `matrix.Mul(a,b)` | same: |`a @ b` | `a * b` | matrix multiply |
| `tensor.Transpose(a)` | <- or `a.T` |`a.transpose()` or `a.T` | `a.'` | transpose of `a` |
| TODO: |  |`a.conj().transpose() or a.conj().T` | `a'` | conjugate transpose of `a` |
| `matrix.Det(a)` | `matrix.Det(a)` | `np.linalg.det(a)` | ? | determinant of `a` |
| `matrix.Identity(3)` | <- |`np.eye(3)` | `eye(3)` | 3x3 identity matrix |
| `matrix.Diagonal(a)` | <- |`np.diag(a)` | `diag(a)` | returns a vector of the diagonal elements of 2D tensor, `a`. Goal returns a read / write view. |
|  |  |`np.diag(v, 0)` | `diag(v,0)` | returns a square diagonal matrix whose nonzero values are the elements of vector, v |
| `matrix.Trace(a)` | <- |`np.trace(a)` | `trace(a)` | returns the sum of the elements along the diagonal of `a`. |
| `matrix.Tri()` | <- |`np.tri()` | `tri()` | returns a new 2D Float64 matrix with 1s in the lower triangular region (including the diagonal) and the remaining upper triangular elements zero |
| `matrix.TriL(a)` | <- |`np.tril(a)` | `tril(a)` | returns a copy of `a` with the lower triangular elements (including the diagonal) from `a` and the remaining upper triangular elements zeroed out |
| `matrix.TriU(a)` | <- |`np.triu(a)` | `triu(a)` | returns a copy of `a` with the upper triangular elements (including the diagonal) from `a` and the remaining lower triangular elements zeroed out |
|  |  |`linalg.inv(a)` | `inv(a)` | inverse of square 2D tensor a |
|  |  |`linalg.pinv(a)` | `pinv(a)` | pseudo-inverse of 2D tensor a |
|  |  |`np.linalg.matrix_rank(a)` | `rank(a)` | matrix rank of a 2D tensor a |
|  |  |`linalg.solve(a, b)` if `a` is square; `linalg.lstsq(a, b)` otherwise | `a\b` | solution of `a x = b` for x |
|  |  |Solve `a.T x.T = b.T` instead | `b/a` | solution of x a = b for x |
|  |  |`U, S, Vh = linalg.svd(a); V = Vh.T` | `[U,S,V]=svd(a)` | singular value decomposition of a |
|  |  |`linalg.cholesky(a)` | `chol(a)` | Cholesky factorization of a 2D tensor |
|  |  |`D,V = linalg.eig(a)` | `[V,D]=eig(a)` | eigenvalues and eigenvectors of `a`, where `[V,D]=eig(a,b)` eigenvalues and eigenvectors of `a, b` where |
|  |  |`D,V = eigs(a, k=3)`  | `D,V = linalg.eig(a, b)` |  `[V,D]=eigs(a,3)` | find the k=3 largest eigenvalues and eigenvectors of 2D tensor, a |
|  |  |`Q,R = linalg.qr(a)`  | `[Q,R]=qr(a,0)` | QR decomposition
|  |  |`P,L,U = linalg.lu(a)` where `a == P@L@U`  | `[L,U,P]=lu(a)` where `a==P'*L*U` | LU decomposition with partial pivoting (note: P(MATLAB) == transpose(P(NumPy))) | 
|  |  |`x = linalg.lstsq(Z, y)` | `x = Z\y` | perform a linear regression of the form |

## Statistics

| `tensor` Go  |   Goal      | NumPy  | MATLAB | Notes            |
| ------------ | ----------- | ------ | ------ | ---------------- |
| `a.max()` or `max(a)` or `stats.Max(a)` | `a.max()` or `np.nanmax(a)` | `max(max(a))` | maximum element of `a`, Goal always ignores `NaN` as missing data |
|  |  |`a.max(0)` | `max(a)` | maximum element of each column of tensor `a` |
|  |  |`a.max(1)` | `max(a,[],2)` | maximum element of each row of tensor `a` |
|  |  |`np.maximum(a, b)` | `max(a,b)` | compares a and b element-wise, and returns the maximum value from each pair |
| `stats.L2Norm(a)` | `np.sqrt(v @ v)` or `np.linalg.norm(v)` | `norm(v)` | L2 norm of vector v |
|  |  |`cg`  | `conjgrad` | conjugate gradients solver |

## FFT and complex numbers

todo: huge amount of work needed to support complex numbers throughout!

| `tensor` Go  |   Goal      | NumPy  | MATLAB | Notes            |
| ------------ | ----------- | ------ | ------ | ---------------- |
|  |  |`np.fft.fft(a)` | `fft(a)` | Fourier transform of `a` |
|  |  |`np.fft.ifft(a)` | `ifft(a)` | inverse Fourier transform of `a` |
|  |  |`signal.resample(x, np.ceil(len(x)/q))` |  `decimate(x, q)` | downsample with low-pass filtering |

## datafs

The [datafs](../tensor/datafs) data filesystem provides a global filesystem-like workspace for storing tensor data, and Goal has special commands and functions to facilitate interacting with it. In an interactive `goal` shell, when you do `##` to switch into math mode, the prompt changes to show your current directory in the datafs, not the regular OS filesystem, and the final prompt character turns into a `#`.

Use `get` and `set` (aliases for `datafs.Get` and `datafs.Set`) to retrieve and store data in the datafs:

* `x := get("path/to/item")` retrieves the tensor data value at given path, which can then be used directly in an expression or saved to a new variable as in this example.

* `set("path/to/item", x)` saves tensor data to given path, overwriting any existing value for that item if it already exists, and creating a new one if not. `x` can be any data expression.

You can use the standard shell commands to navigate around the data filesystem:

* `cd <dir>` to change the current working directory. By default, new variables created in the shell are also recorded into the current working directory for later access.

* `ls [-l,r] [dir]` list the contents of a directory; without arguments, it shows the current directory. The `-l` option shows each element on a separate line with its shape. `-r` does a recursive list through subdirectories.

* `mkdir <dir>` makes a new subdirectory.

TODO: other commands, etc.



