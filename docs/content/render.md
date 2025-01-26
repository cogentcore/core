+++
Categories = ["Architecture"]
+++

**Rendering** is the process of converting [[widget]]s into images uploaded to a window. This page documents low-level [[architecture]] details of rendering.

Almost all of rendering is pure Go and does **not** depend on a WebView or any other system APIs; instead, [[scene]]s are rendered to images using custom logic written in Go (see [[doc:paint]]). The rendered images are then combined as necessary before being uploaded to a window. Only at the final stage of uploading to the system window does Cogent Core using system APIs. Given that almost all of the rendering process is platform-independent, widgets look the same on all platforms.

## Rendering logic

At the highest level, rendering is made robust by having a completely separate, mutex lock-protected pass where all render-level updating takes place.  This render pass is triggered by [[doc:events.WindowPaint]] events that are sent regularly at 60 FPS (frames per second).  If nothing needs to be updated, nothing happens (which is the typical case for most frames), so it is not a significant additional cost.

The usual processing of events that arise in response to user GUI actions, or any other source of changes, sets flags that determine what kind of updating needs to happen during rendering.  These are typically set via [[doc:core.WidgetBase.NeedsRender]] or [[doc:core.WidgetBase.NeedsLayout]] calls.

The first step in the `renderWindow.renderWindow()` rendering function is to call `updateAll()` which ends up calling `doUpdate()` on the [[doc:core.Scene]] elements within a render window, and this function is what checks if a new [layout](layout) pass has been called for, or whether any individual widgets need to be rendered. This rendering update writes to a separate `image.RGBA` owned by the Scene, which provides the raw input for the final image rendered to the window.

Most updating of widgets happens in the event processing loop, which is synchronous (one event is processed at a time).  

For any updating that happens outside of the normal event loop (e.g., timer-based animations etc), you must go through [[doc:WidgetBase.AsyncLock]] and [[doc:WidgetBase.AsyncUnlock]].

## Structure of the renderWindow

The overall management of rendering is organized as follows:

* `renderWindow` uses the [[doc:system.Drawer]] interface (which is implemented using WebGPU on desktop, mobile, and web) to composite and "blit" (quickly render) all the individual image elements out to the actual hardware window that you see, layered in the proper order.  On the web and offscreen (testing) platforms, Drawer is implemented using Go `image.Draw` functions, with no other hardware dependencies.

* `stages` in the renderWindow manages a stack of `stage` elements, each of which manages an individual [[doc:core.Scene]].  There are different types of stage, specified by the [[doc:core.StageTypes]] enum, including the main `WindowStage` for primary app content, `DialogStage`, `MenuStage`, `TooltipStage` etc.

* The `stage` also manages a collection of [[doc:core.Sprites]] which provide lightweight graphical elements that are rendered over the top of the underlying content, for things like selection boxes, cursors, etc.

* The [[doc:core.Scene]] is the base-level element that contains a coherent set of Widget elements, which are subject to mutual layout constraints and are all rendered onto a shared image.  It specifies the _content_ of a GUI element, whereas the stage has parameters that control the overall behavior and event processing, which are different for popups vs main windows.

