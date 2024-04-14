# jsfs

Package jsfs provides a Node.js style filesystem API in Go that can be used to allow os functions to work on wasm. It implements all standard Node.js fs functions except for the synchronous versions, as those are not needed by the Go standard library.

It is built on [hackpadfs](https://github.com/hack-pad/hackpadfs) and based on [hackpad](https://github.com/hack-pad/hackpad), which are licensed under the [Apache 2.0 License](https://github.com/hack-pad/hackpad/blob/main/LICENSE).
