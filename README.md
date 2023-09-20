# goosi

Goosi provides a Go Operating System Interface framework to support events, window management, and other OS-specific functionality needed for full GUI support.

The following implementations of the higher-level goosi interfaces are supported:

* `desktop` uses `glfw` and other OS-specific code to support Windows, MacOS, X11, and Wayland.
* `ios` supports the Apple iOS mobile platform.
* `android` supports the Android mobile platform.
* `web` browsers will be supported in the future.

The `Platform` type enumerates these supported platforms and should be used to conditionalize behavior, instead of the GOOS value typically used in Go.

