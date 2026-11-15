# Testing

## Running the tests

The CI (`.github/workflows/go.yml`) runs all of the packages that build on Linux:

```sh
go test -short ./...
```

On the CI, Mesa's software GPU (lavapipe) is installed for the tests of `gpu` and `xyz`, and the tests run in an Xvfb display. The tests that need a GPU call `gpu.SkipIfNoGPU(t)`, which skips them on a machine without a GPU adapter, unless `CORE_REQUIRE_GPU` is set (as on the CI), which makes them fail instead.

The `video` tests need FFmpeg (see go-astiav), which the CI builds.

## Widget tests

Widget tests make a body, and assert its rendered image with `core.AssertRender`, optionally after functions that interact with it, each of which is followed by the handling of all of the events that it causes:

```go
b := core.NewBody()
tf := core.NewTextField(b)
core.AssertRender(t, b, "text-field/typed", core.Locked(b, func() {
	tf.SetFocus()
	core.TypeText(tf, "hello")
	core.SendKeyFunction(tf, keymap.WordLeft)
	core.SendKey(tf, 0, key.CodeRightArrow, key.Shift)
}))
```

- `core.Locked(b, fun)` calls `fun` with the AsyncLock of the body held, which is needed to change widgets from a test.
- `core.SendKey` sends a key chord with modifier flags, `core.TypeText` types text, and `core.SendKeyFunction` sends the key chord of a key function in the active key map, so that tests do not depend on the platform.
- `core.RenderImage` is the rendering part of `AssertRender`, for tests that check things other than the image.
- The offscreen driver has an in-memory clipboard, so copy and paste can be tested.

## Test images

The images of the tests are in the `testdata` directories of the packages, and are not in the repository, except for some that are (as the snapshots of htmlcore). The CI keeps its images as an artifact. A test with a new image saves it on its first run.

To update the images after a change to rendering, run the tests with the `update` build tag. Images with exactly the same pixels are kept as they are.

```sh
go test -tags update -run TestName ./core/
```

## Platforms

The tests above run on the desktop platforms. Others are run locally, when working on those areas:

- web: `CORE_GOSCRIPT_TEST=1 go test -run TestGoScript ./core/` runs the core tests in Chrome, built with goscript.
- iOS and Android: `core run iossimulator` and `core run android` run an app in the simulator or emulator.
- The desktop driver (`system/driver/desktop`) has tests for each desktop platform, with build tags (`desktop_darwin_test.go`, `desktop_linux_test.go`, `desktop_windows_test.go`, and the shared `desktop_test.go`), whose `TestMain` runs the main loop on the main thread:
  - macOS: `go test -short ./system/...` opens a real window, and does a clipboard round trip that restores the clipboard.
  - Linux: `xvfb-run -a go test -short ./system/... ./video/...` (as on the CI), or with a real X or Wayland session. The window and clipboard tests skip without `DISPLAY` or `WAYLAND_DISPLAY`, and the clipboard test only runs with `CI` set (the X11 clipboard cannot tell whether it is safe to overwrite).
  - Windows: `go test -short ./system/...` in a VM. Set `CORE_NO_GUI_TESTS=1` when running over ssh or in a service, which skips the window tests.
