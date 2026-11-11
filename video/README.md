# video

A video player component for the Cogent Core GUI framework

On desktop platforms, video is decoded with [FFmpeg](https://ffmpeg.org) through [go-astiav](https://github.com/asticode/go-astiav), which requires the FFmpeg development libraries, found with `pkg-config`. The go-astiav version in `go.mod` requires FFmpeg 9.0 (for example, `brew install ffmpeg` on macOS). Decoding video is not yet supported on the web.

Apps that do not need video can be built without the FFmpeg libraries with the `novideo` build tag (`go build -tags novideo`), with which `OpenMedia` returns an error.
