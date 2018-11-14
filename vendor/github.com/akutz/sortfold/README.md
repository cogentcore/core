# SortFold
SortFold is a Go package that enables sorting string slices in increasing
order using a case-insensitive comparison.

## Sorting String Slices
The standard library's `sort` package provides the ability to sort
slices of arbitrary types in increasing order, but when it comes to
strings, Go can be a little sensitive. For example:
([run](https://play.golang.org/p/JZN4iokIMD)):

```go
package main

import (
	"fmt"
	"sort"
)

func main() {
	data := []string{"A", "b", "c", "D"}
	fmt.Println(sort.StringsAreSorted(data))

	sort.Strings(data)
	fmt.Println(data)

	fmt.Println(sort.StringsAreSorted(data))
}
```

```
false
[A D b c]
true
```

While `A b c D` is sorted if elements are compared in a case-insensitive
manner, the `sort` package's string sorting is case-sensitive. Luckily,
the `sort` package also makes it trivial to provide custom comparison
logic.

## Sorting Sans Sensitivity
The `sort` package defines functions that accept a custom comparator
function in order to implement custom sorting. The following example
illustrates how to use these functions to provide a case-insensitive
sort ([run](https://play.golang.org/p/Es8SP_r6OA)):

```go
package main

import (
	"fmt"
	"sort"
	"strings"
)

func main() {
	data := []string{"A", "b", "c", "D"}

	less := func(i, j int) bool {
		return strings.ToLower(data[i]) < strings.ToLower(data[j])
	}

	fmt.Println(sort.SliceIsSorted(data, less))
}
```

```
true
```

The above example showcases the use of `sort.SliceIsSorted` and the
comparator function `less` to determine the slice `data` is already
sorted in a case-insensitive manner. While this may seem like the end
of the story, the above implementation comes with a cost.

## Performance Concerns
The Go [`sort.Sort`](https://golang.org/pkg/sort/#Sort) function is an
implementation of the [quicksort](https://en.wikipedia.org/wiki/Quicksort)
algorithm. This means at best a call to `sort.Sort` is `O(n*log(n))` complex
and at worst is `O(n^2)`. In other words, in an ideal scenario the sort
operation will make `n*log(n)` calls to `Less` and in a worst case situation
`Less` will be invoked `n^2` times.

The example in the previous section uses
[`strings.ToLower`](https://golang.org/pkg/strings/#ToLower) to standardize
the case of both operands prior to comparison. Because `string` values in
Go are immutable, a function like `ToLower` allocates new memory to hold
the result of the operation.

Thus, in a best-case scenario, `2(n*log(n))` new strings are allocated
when sorting slices of strings in a case-insensitive manner with
`strings.ToLower` and up to `2(n^2)` new strings are created at worst.
Obviously this solution does not scale.

## Comparing Case Folding
The reason `strings.ToLower` (or
[`strings.ToUpper`](https://golang.org/pkg/strings/#ToUpper)) is required
in the first place is due to Go's lack of a function that uses
[Unicode case-folding](https://www.w3.org/International/wiki/Case_folding)
to indicate if string `a` is less than string `b`. The Go standard library
includes the following functions for comparing strings:

| Function | Case-Folding | Description |
|----------|:--------------:|-------------|
| [`strings.Compare`](https://golang.org/pkg/strings/#Compare) | | Returns `1` if `a` < `b`, `0` if `a` == `b`, `1` if `a` > `b` |
| [`strings.EqualFold`](https://golang.org/pkg/strings/#EqualFold) | ✓ | Returns `true` if two strings are equal |

This package introduces `sortfold.CompareFold`, a function that compares
strings using Unicode case-folding like `strings.EqualFold` and behaves
like `strings.Compare` with respect to the return value:

```go
package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/akutz/sortfold"
)

func main() {
	data := []string{"A", "b", "c", "D"}

	less := func(i, j int) bool {
		return sortfold.CompareFold(data[i], data[j]) < 0
	}

	fmt.Println(sort.SliceIsSorted(data, less))
}
```

```
true
```

## Performance Comparison
The introduction of `sortfold.CompareFold` provides the ability to sort
slices of strings in case-insensitive manner that is highly performant.
The benchmark results below illustrate the difference between using two
implementations of `sort.Interface` to sort string slices of varying
length. The benchmarks that have the prefix `Benchmark_FoldedSort` use
`sortfold.StringSlice`, which itself uses `sortfold.CompareFold`. The
benchmarks with the prefix `Benchmark_LCasedSort` convert strings to
lowercase prior to comparison:

### Charted Data

| Benchmark | Chart |
|-----------|-------|
| 2 Chars | ![2 Chars](https://imgur.com/dy75LWT.png) |
| 26 Chars | ![26 Chars](https://imgur.com/4VdonVI.png) |
| 542 Words | ![542 Words](https://imgur.com/W5HYmmI.png) |
| 54,200 Words | ![54200 Words](https://imgur.com/rrnihFU.png) |
| 542,000 Words | ![542000 Words](https://imgur.com/l5lmVGQ.png) |

### Raw Data

```shell
$ go test -benchmem -run Bench -bench . -v
goos: darwin
goarch: amd64
pkg: github.com/akutz/sortfold
Benchmark_FoldedSort______2_Chars____LowerCase_Sorted-8     	30000000	        40.5 ns/op	       0 B/op	       0 allocs/op
Benchmark_LCasedSort______2_Chars____LowerCase_Sorted-8     	30000000	        49.3 ns/op	       0 B/op	       0 allocs/op
Benchmark_FoldedSort______2_Chars____LowerCase_Shuffled-8   	30000000	        46.4 ns/op	       0 B/op	       0 allocs/op
Benchmark_LCasedSort______2_Chars____LowerCase_Shuffled-8   	20000000	        59.7 ns/op	       0 B/op	       0 allocs/op
Benchmark_FoldedSort______2_Chars____MixedCase_Sorted-8     	50000000	        37.9 ns/op	       0 B/op	       0 allocs/op
Benchmark_LCasedSort______2_Chars____MixedCase_Sorted-8     	10000000	       114 ns/op	       8 B/op	       2 allocs/op
Benchmark_FoldedSort______2_Chars__1_MixedCase_Shuffled-8   	50000000	        39.4 ns/op	       0 B/op	       0 allocs/op
Benchmark_LCasedSort______2_Chars__1_MixedCase_Shuffled-8   	10000000	       113 ns/op	       8 B/op	       2 allocs/op
Benchmark_FoldedSort_____26_Chars____LowerCase_Sorted-8     	  500000	      2889 ns/op	       0 B/op	       0 allocs/op
Benchmark_LCasedSort_____26_Chars____LowerCase_Sorted-8     	  500000	      2584 ns/op	       0 B/op	       0 allocs/op
Benchmark_FoldedSort_____26_Chars____LowerCase_Shuffled-8   	  500000	      2803 ns/op	       0 B/op	       0 allocs/op
Benchmark_LCasedSort_____26_Chars____LowerCase_Shuffled-8   	  500000	      4112 ns/op	       0 B/op	       0 allocs/op
Benchmark_FoldedSort_____26_Chars__1_MixedCase_Sorted-8     	  500000	      2694 ns/op	       0 B/op	       0 allocs/op
Benchmark_LCasedSort_____26_Chars__1_MixedCase_Sorted-8     	  500000	      2677 ns/op	      40 B/op	      10 allocs/op
Benchmark_FoldedSort_____26_Chars__1_MixedCase_Shuffled-8   	  500000	      2821 ns/op	       0 B/op	       0 allocs/op
Benchmark_LCasedSort_____26_Chars__1_MixedCase_Shuffled-8   	  300000	      5272 ns/op	      72 B/op	      18 allocs/op
Benchmark_FoldedSort_____26_Chars__5_MixedCase_Sorted-8     	 1000000	      1647 ns/op	       0 B/op	       0 allocs/op
Benchmark_LCasedSort_____26_Chars__5_MixedCase_Sorted-8     	  500000	      3448 ns/op	     160 B/op	      40 allocs/op
Benchmark_FoldedSort_____26_Chars__5_MixedCase_Shuffled-8   	 1000000	      2594 ns/op	       0 B/op	       0 allocs/op
Benchmark_LCasedSort_____26_Chars__5_MixedCase_Shuffled-8   	  300000	      5918 ns/op	     280 B/op	      70 allocs/op
Benchmark_FoldedSort_____26_Chars_10_MixedCase_Sorted-8     	 1000000	      1496 ns/op	       0 B/op	       0 allocs/op
Benchmark_LCasedSort_____26_Chars_10_MixedCase_Sorted-8     	  300000	      4417 ns/op	     328 B/op	      82 allocs/op
Benchmark_FoldedSort_____26_Chars_10_MixedCase_Shuffled-8   	 1000000	      2113 ns/op	       0 B/op	       0 allocs/op
Benchmark_LCasedSort_____26_Chars_10_MixedCase_Shuffled-8   	  200000	      7425 ns/op	     536 B/op	     134 allocs/op
Benchmark_FoldedSort____542_Words____MixedCase_Sorted-8     	   10000	    149475 ns/op	       0 B/op	       0 allocs/op
Benchmark_LCasedSort____542_Words____MixedCase_Sorted-8     	    3000	    477477 ns/op	   20400 B/op	    2046 allocs/op
Benchmark_FoldedSort____542_Words____MixedCase_Shuffled-8   	   10000	    195502 ns/op	       0 B/op	       0 allocs/op
Benchmark_LCasedSort____542_Words____MixedCase_Shuffled-8   	    3000	    542718 ns/op	   21856 B/op	    2208 allocs/op
Benchmark_FoldedSort__54200_Words____MixedCase_Shuffled-8   	     100	  16854892 ns/op	       0 B/op	       0 allocs/op
Benchmark_LCasedSort__54200_Words____MixedCase_Shuffled-8   	      30	  53410004 ns/op	 2307312 B/op	  216820 allocs/op
Benchmark_FoldedSort_542000_Words____MixedCase_Shuffled-8   	      10	 176222982 ns/op	       0 B/op	       0 allocs/op
Benchmark_LCasedSort_542000_Words____MixedCase_Shuffled-8   	       2	 545716283 ns/op	30080968 B/op	 3180838 allocs/op
PASS
ok  	github.com/akutz/sortfold	93.355s
```
