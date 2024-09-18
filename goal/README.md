# Goal: Go augmented language

_Goal_ is an augmented version of the _Go_ language, which combines the best parts of _Go_, `bash`, and Python, to provide and integrated shell and numerical expression processing experience.

_Goal_ transpiles directly into Go, so it automatically leverages all the great features of Go, and remains fully compatible with it.  The augmentation is designed to overcome some of the limitations of Go in specific domains:

* Shell scripting, where you want to be able to directly call other executable programs with arguments, without having to navigate all the complexity of the standard [os.exec](https://pkg.go.dev/os/exec) package.

* Numerical / math / data processing, where you want to be able to write simple mathematical expressions operating on vectors, matricies and other more powerful data types, without having to constantly worry about type conversions and iterators etc. Python is the dominant language here precisely because it lets you ignore type information and write such expressions.

The main goal of _Goal_ is to achieve a "best of both worlds" solution that retains all the type safety and explicitness of Go for all the surrounding control flow and large-scale application logic, while also allowing for a more relaxed syntax in specific, well-defined domains where the Go language has been a barrier.  Thus, unlike Python where there are various weak attempts to try to encourage better coding habits, _Goal_ retains in its _Go_ foundation a fundamentally scalable, "industrial strength" language that has already proven its worth in countless real-world applications.

For the shell scripting aspect of _Goal_, the simple idea is that each line of code is either Go or shell commands, determined in a fairly intuitive way mostly by the content at the start of the line (formal rules below). If a line starts off with something like `ls -la...` then it is clear that it is not valid Go code, and it is therefore processed as a shell command.

You can intermix Go within a shell line by wrapping an expression with `{ }` braces, and a Go expression can contain shell code by using backticks (\`) (raw quoted strings use double-backticks \`\`).  Here's an example:
```go
for i, f := range goal.SplitLines(`ls -la`) {  // `ls` executes, returns string
    echo {i} {strings.ToLower(f)}               // {} surrounds Go within shell
}
```
where `goal.SplitLines` is a function that runs `strings.Split(arg, "\n")`, defined in the `goal` standard library of such frequently-used helper functions.

For the mathematical expressions, we use `$` symbols (as in markdown) to demarcate such expressions:
```go
for _, x := range $[1,2,3]$ {
    fmt.Println($x^2$)
}
```

In general, _Goal_ is designed to be as compatible with Python numpy / scipy syntax as possible, while also adding a few Go-specific additions as well.  All elements of a _Goal_ math expression are [tensors](../tensor), specifically `*tensor.Indexed`, which can represent everything from a scalar to an n-dimenstional tensor.  These are called an "array" in numpy terms.

We henceforth refer to shell code as `exec` code (in reference to the Go & Cogent `exec` package that we use to execute programs), given the potential ambituity of the entire `cosh` language being the shell. There are different syntactic formatting rules for these two domains of Go and Exec, within cosh:

* Go code is processed and formatted as usual (e.g., white space is irrelevant, etc).
* Exec code is space separated, like normal command-line invocations.

You can easily perform handy duration and data size formatting:

```go
22010706 * time.Nanosecond  // 22.010706ms
datasize.Size(44610930)     // 42.5 MB
```

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

# Script Files and Makefile-like functionality

As with most scripting languages, a file of cosh code can be made directly executable by appending a "shebang" expression at the start of the file:

```sh
#!/usr/bin/env cosh
```

When executed this way, any additional args are available via an `args []any` variable, which can be passed to a command as follows:
```go
install {args...}
```
or by referring to specific arg indexes etc.

To make a script behave like a standard Makefile, you can define different `command`s for each of the make commands, and then add the following at the end of the file to use the args to run commands:

```go
shell.RunCommands(args)
```

See [make](cmd/cosh/testdata/make) for an example, in `cmd/cosh/testdata/make`, which can be run for example using:

```sh
./make build
```

Note that there is nothing special about the name `make` here, so this can be done with any file.

The `make` package defines a number of useful utility functions that accomplish the standard dependency and file timestamp checking functionality from the standard `make` command, as in the [magefile](https://magefile.org/dependencies/) system.  Note that the cosh direct exec command syntax makes the resulting make files much closer to a standard bash-like Makefile, while still having all the benefits of Go control and expressions, compared to magefile.

TODO: implement and document above.

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

# Math Expression Details

In general, _Goal_ is designed to be as compatible with Python numpy / scipy syntax as possible, while also adding a few Go-specific additions as well.  The `np.` prefix on numpy global functions is optional, and corresponding field-like properties of tensors turn into the appropriate methods during the transpiling process.

All elements of a _Goal_ math expression are [tensors](../tensor), specifically `*tensor.Indexed`, which can represent everything from a scalar to an n-dimenstional tensor.  These are called an "array" in numpy terms.

Here's a full list of equivalents, from [numpy-for-matlab-users](https://numpy.org/doc/stable/user/numpy-for-matlab-users.html)

| Goal  | Python | MATLAB | Notes  |
| ----- | ------ | ------ | ------ |
| same: | `np.ndim(a)` or `a.ndim`   | `ndims(a)` | number of dimensions of array `a` |
| `len(a)` or `a.len` or: | `np.size(a)` or `a.size`   | `numel(a)` | number of elements of array `a` |
| same: | `np.shape(a)` or `a.shape` | `size(a)`  | "size" of each dimension in a; `shape` returns a 1D `int` array |
| same: | `a.shape[n-1]` | `size(a,n)` | the number of elements of the n-th dimension of array a |
| `tensor([[1., 2., 3.], [4., 5., 6.]])` or: | `(np.array([[1., 2., 3.], [4., 5., 6.]])` | `[ 1 2 3; 4 5 6 ]` | define a 2x3 2D array |
| same: | `np.block([[a, b], [c, d]])` | `[ a b; c d ]` | construct a matrix from blocks a, b, c, and d |
| same: | `a[-1]` | `a(end)` | access last element |
| same: | `a[1, 4]` | `a(2,5)` | access element in second row, fifth column in 2D array a |
| same: | `a[1]` or `a[1, :]` | `a(2,:)` | entire second row of 2D array a; unspecified dimensions are equivalent to `:` |
| same: | `a[0:5]` or `a[:5]` or `a[0:5, :]` | `a(1:5,:)` | same as Go slice ranging |
| same: | `a[-5:]` | `a(end-4:end,:)` | first 5 rows of 2D array a |
| same: | `a[0:3, 4:9]` | `a(1:3,5:9)` | last 5 rows of 2D array a |
| same: | `a[np.ix_([1, 3, 4], [0, 2])]` | `a([2,4,5],[1,3])` | The first through third rows and fifth through ninth columns of a 2D array, a. |


rows 2,4 and 5 and columns 1 and 3. This allows the matrix to be modified, and doesn’t require a regular slice.

a(3:2:21,:)

a[2:21:2,:]

every other row of a, starting with the third and going to the twenty-first

a(1:2:end,:)

a[::2, :]

every other row of a, starting with the first

a(end:-1:1,:) or flipud(a)

a[::-1,:]

a with rows in reverse order

a([1:end 1],:)

a[np.r_[:len(a),0]]

a with copy of the first row appended to the end

a.'

a.transpose() or a.T

transpose of a

a'

a.conj().transpose() or a.conj().T

conjugate transpose of a

a * b

a @ b

matrix multiply

a .* b

a * b

element-wise multiply

a./b

a/b

element-wise divide

a.^3

a**3

element-wise exponentiation

(a > 0.5)

(a > 0.5)

matrix whose i,jth element is (a_ij > 0.5). The MATLAB result is an array of logical values 0 and 1. The NumPy result is an array of the boolean values False and True.

find(a > 0.5)

np.nonzero(a > 0.5)

find the indices where (a > 0.5)

a(:,find(v > 0.5))

a[:,np.nonzero(v > 0.5)[0]]

extract the columns of a where vector v > 0.5

a(:,find(v>0.5))

a[:, v.T > 0.5]

extract the columns of a where column vector v > 0.5

a(a<0.5)=0

a[a < 0.5]=0

a with elements less than 0.5 zeroed out

a .* (a>0.5)

a * (a > 0.5)

a with elements less than 0.5 zeroed out

a(:) = 3

a[:] = 3

set all values to the same scalar value

y=x

y = x.copy()

NumPy assigns by reference

y=x(2,:)

y = x[1, :].copy()

NumPy slices are by reference

y=x(:)

y = x.flatten()

turn array into vector (note that this forces a copy). To obtain the same data ordering as in MATLAB, use x.flatten('F').

1:10

np.arange(1., 11.) or np.r_[1.:11.] or np.r_[1:10:10j]

create an increasing vector (see note RANGES)

0:9

np.arange(10.) or np.r_[:10.] or np.r_[:9:10j]

create an increasing vector (see note RANGES)

[1:10]'

np.arange(1.,11.)[:, np.newaxis]

create a column vector

zeros(3,4)

np.zeros((3, 4))

3x4 two-dimensional array full of 64-bit floating point zeros

zeros(3,4,5)

np.zeros((3, 4, 5))

3x4x5 three-dimensional array full of 64-bit floating point zeros

ones(3,4)

np.ones((3, 4))

3x4 two-dimensional array full of 64-bit floating point ones

eye(3)

np.eye(3)

3x3 identity matrix

diag(a)

np.diag(a)

returns a vector of the diagonal elements of 2D array, a

diag(v,0)

np.diag(v, 0)

returns a square diagonal matrix whose nonzero values are the elements of vector, v

rng(42,'twister')
rand(3,4)
from numpy.random import default_rng
rng = default_rng(42)
rng.random(3, 4)
or older version: random.rand((3, 4))

generate a random 3x4 array with default random number generator and seed = 42

linspace(1,3,4)

np.linspace(1,3,4)

4 equally spaced samples between 1 and 3, inclusive

[x,y]=meshgrid(0:8,0:5)

np.mgrid[0:9.,0:6.] or np.meshgrid(r_[0:9.],r_[0:6.])

two 2D arrays: one of x values, the other of y values

ogrid[0:9.,0:6.] or np.ix_(np.r_[0:9.],np.r_[0:6.]

the best way to eval functions on a grid

[x,y]=meshgrid([1,2,4],[2,4,5])

np.meshgrid([1,2,4],[2,4,5])

np.ix_([1,2,4],[2,4,5])

the best way to eval functions on a grid

repmat(a, m, n)

np.tile(a, (m, n))

create m by n copies of a

[a b]

np.concatenate((a,b),1) or np.hstack((a,b)) or np.column_stack((a,b)) or np.c_[a,b]

concatenate columns of a and b

[a; b]

np.concatenate((a,b)) or np.vstack((a,b)) or np.r_[a,b]

concatenate rows of a and b

max(max(a))

a.max() or np.nanmax(a)

maximum element of a (with ndims(a)<=2 for MATLAB, if there are NaN’s, nanmax will ignore these and return largest value)

max(a)

a.max(0)

maximum element of each column of array a

max(a,[],2)

a.max(1)

maximum element of each row of array a

max(a,b)

np.maximum(a, b)

compares a and b element-wise, and returns the maximum value from each pair

norm(v)

np.sqrt(v @ v) or np.linalg.norm(v)

L2 norm of vector v

a & b

logical_and(a,b)

element-by-element AND operator (NumPy ufunc) See note LOGICOPS

a | b

np.logical_or(a,b)

element-by-element OR operator (NumPy ufunc) See note LOGICOPS

bitand(a,b)

a & b

bitwise AND operator (Python native and NumPy ufunc)

bitor(a,b)

a | b

bitwise OR operator (Python native and NumPy ufunc)

inv(a)

linalg.inv(a)

inverse of square 2D array a

pinv(a)

linalg.pinv(a)

pseudo-inverse of 2D array a

rank(a)

np.linalg.matrix_rank(a)

matrix rank of a 2D array a

a\b

linalg.solve(a, b) if a is square; linalg.lstsq(a, b) otherwise

solution of a x = b for x

b/a

Solve a.T x.T = b.T instead

solution of x a = b for x

[U,S,V]=svd(a)

U, S, Vh = linalg.svd(a); V = Vh.T

singular value decomposition of a

chol(a)

linalg.cholesky(a)

Cholesky factorization of a 2D array

[V,D]=eig(a)

D,V = linalg.eig(a)

eigenvalues 
 and eigenvectors 
 of a, where 

[V,D]=eig(a,b)

D,V = linalg.eig(a, b)

eigenvalues 
 and eigenvectors 
 of a, b where 

[V,D]=eigs(a,3)

D,V = eigs(a, k=3)

find the k=3 largest eigenvalues and eigenvectors of 2D array, a

[Q,R]=qr(a,0)

Q,R = linalg.qr(a)

QR decomposition

[L,U,P]=lu(a) where a==P'*L*U

P,L,U = linalg.lu(a) where a == P@L@U

LU decomposition with partial pivoting (note: P(MATLAB) == transpose(P(NumPy)))

conjgrad

cg

conjugate gradients solver

fft(a)

np.fft.fft(a)

Fourier transform of a

ifft(a)

np.fft.ifft(a)

inverse Fourier transform of a

sort(a)

np.sort(a) or a.sort(axis=0)

sort each column of a 2D array, a

sort(a, 2)

np.sort(a, axis=1) or a.sort(axis=1)

sort the each row of 2D array, a

[b,I]=sortrows(a,1)

I = np.argsort(a[:, 0]); b = a[I,:]

save the array a as array b with rows sorted by the first column

x = Z\y

x = linalg.lstsq(Z, y)

perform a linear regression of the form 

decimate(x, q)

signal.resample(x, np.ceil(len(x)/q))

downsample with low-pass filtering

unique(a)

np.unique(a)

a vector of unique values in array a

squeeze(a)

a.squeeze()

remove singleton dimensions of array a. Note that MATLAB will always return arrays of 2D or higher while NumPy will return arrays of 0D or higher


