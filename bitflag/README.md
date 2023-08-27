# bitflag

[![Go Report Card](https://goreportcard.com/badge/goki.dev/ki/bitflag)](https://goreportcard.com/report/goki.dev/ki/bitflag)
[![GoDoc](https://godoc.org/goki.dev/ki/bitflag?status.svg)](http://godoc.org/goki.dev/ki/bitflag)

Package `bitflag` provides simple bit flag setting, checking, and clearing
methods that take bit position args as ints (from const int eunum iota's)
and do the bit shifting from there -- although a tiny bit slower, the
convenience of maintaining ordinal lists of bit positions greatly outweighs
that cost -- see kit type registry for further enum management functions
