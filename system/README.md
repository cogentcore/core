# system

System provides a Go operating system interface framework to support events, window management, and other OS-specific functionality needed for full GUI support.

The following implementations of the higher-level system interfaces are supported (see the [driver directory](./driver)):

* `desktop` uses `glfw` and other OS-specific code to support Windows, MacOS, X11, and Wayland.
* `ios` supports the Apple iOS mobile platform.
* `android` supports the Android mobile platform.
* `web` supports running apps on web browsers through WASM.
* `offscreen` supports running apps detached from any outside platform.

The `Platform` type enumerates these supported platforms and should be used to conditionalize behavior, instead of the GOOS value typically used in Go.

## IMPORTANT

After making any changes to `GoNativeActivity.java` in the Android driver, you **need** to run go generate in [../cmd/mobile](../cmd/mobile) and reinstall the core tool.
