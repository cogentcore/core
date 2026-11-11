# video

A video player component for the Cogent Core GUI framework

On desktop platforms, video is decoded with [FFmpeg](https://ffmpeg.org) through [go-astiav](https://github.com/asticode/go-astiav), which requires the FFmpeg development libraries, found with `pkg-config`. The go-astiav version in `go.mod` requires FFmpeg 9.0, which `core setup -video` installs (with Homebrew on macOS, and as the prebuilt builds of [BtbN](https://github.com/BtbN/FFmpeg-Builds) on Linux and Windows, which the core command builds and runs apps with), and `core update -video` updates. When go-astiav is updated, `ffmpegVersion` in cmd/ffmpeg.go must be updated to its FFmpeg version.

On the web, a `Video` is a video element of the browser, which plays it (see `video_js.go`), and files of the app such as videos are served with it with `Web.Files` in `core.toml` (see examples/video).

Apps that do not need video can be built without the FFmpeg libraries with the `novideo` build tag (`go build -tags novideo`), with which `OpenMedia` returns an error.
