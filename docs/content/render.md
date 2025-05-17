+++
Categories = ["Architecture"]
+++

**Rendering** is the process of converting [[widget]]s (or other sources, such as SVG) into images uploaded to a window. This page documents low-level [[architecture]] details of rendering.

On all platforms except web, the core rendering logic is pure Go, with cgo used for pushing images to the system window. On web, we use HTML canvas, as discussed later.

The overall flow of rendering is:

* Source (SVG, `core.Scene`, etc) -> `Painter` (`render.Render`) -> `render.Renderer` (`image.Image`, SVG, PDF)

All rendering goes through the `Painter` object in the [[doc:paint]] package, which provides a standard set of drawing functions, operating on a `State` that has a stack of `Context` to provide context for these drawing functions (style, transform, bounds, clipping, and mask). Each of these drawing functions is recorded in a `Render` list of render `Item`s (see [[doc:paint/render]]), which provides the intermediate representation of everything that needs to be rendered in a given render update pass. There are three basic types of render `Item`s:

* `Path` encodes all vector graphics operations, which ultimately reduce to only four fundamental drawing functions: `MoveTo`, `LineTo`, `QuadTo`, and `CubeTo`. We adapted the extensive [canvas](https://github.com/tdewolff/canvas) package's Path implementation, which also provides an `ArcTo` primitive that is then compiled down into a `QuadTo`.

* `Text` encodes [[doc:text/shaped]] line(s) of text, which are most often rendered using automatically cached images of rendered font glyph outlines (svg and bitmap representations are also supported).

* `Image` drawing and related raster-based operations are supported by the [[doc:paint/pimage]] package.

This standardized intermediate representation of every rendering step then allows different platforms to actually render the resulting _rasterized_ result (what you actually see on the screen as pixels of different colors) using the most efficient rasterization mechanism for that platform. In addition, this internal representation has sufficient structure to generate `SVG` and `PDF` document outputs, which are typically more compact than a rasterized version, and critically allow further editing etc.

The [[doc:paint/render]] package defines a `Renderer` interface that is implemented by the different types of renderers, which take the `render.Render` from the `paint.Painter` `RenderDone()` method as input and generate the different outputs.

On most platforms (desktop, mobile), the standard rasterization uses our version of the [rasterx](https://github.com/srwiley/rasterx) Go-based rasterizer, which extensive testing has determined to be significantly faster than other Go-based alternatives, and for most GUI rendering cases, is fast enough to not be a major factor in overall updating time. Nevertheless, we will eventually explore a WebGPU based implementation based on the highly optimized [vello](https://github.com/linebender/vello) framework in Rust (see [jello](https://github.com/dominikh/jello) for a Go port).

On the web (`js` platform), we take advantage of the standardized, typically GPU-optimized rendering provided by the html canvas element (see [[doc:paint/renderers/htmlcanvas]]). This is significantly faster than earlier versions of Cogent Core that uploaded Go-rendered images, and provides a hardware-accelerated rendering speed often faster than the desktop platform. This includes text rendering which requires a careful integration of Go and web-based mechanisms to perform internationalized text shaping (see [[doc:text/shaped/shapers/shapedjs]]).

## Core Scene and Widget rendering logic

At the highest level, rendering is made robust by having a completely separate, mutex lock-protected pass where all render-level updating takes place.  This render pass is triggered by [[doc:events.WindowPaint]] events that are sent regularly at the monitor's refresh rate. If nothing needs to be updated, nothing happens (which is the typical case for most frames), so it is not a significant additional cost.

The usual processing of events that arise in response to user GUI actions, or any other source of changes, sets flags that determine what kind of updating needs to happen during rendering.  These are typically set via [[doc:core.WidgetBase.NeedsRender]] or [[doc:core.WidgetBase.NeedsLayout]] calls.

The first step in the `renderWindow.renderWindow()` rendering function is to call `updateAll()` which ends up calling `doUpdate()` on the [[doc:core.Scene]] elements within a render window, and this function is what checks if a new [layout](layout) pass has been called for, or whether any individual widgets need to be rendered.

Most updating of widgets happens in the event processing loop, which is synchronous (one event is processed at a time).  

For any updating that happens outside of the normal event loop (e.g., timer-based animations etc), you must go through [[doc:core.WidgetBase.AsyncLock]] and [[doc:core.WidgetBase.AsyncUnlock]] (see [[async]]).

The result of the `renderWindow()` function for each Scene is a `render.Render` list of rendering commands, which could be just for one widget that needed updating, or for the entire scene if a new layout was needed.

## Composer and sources

The [[doc:system.composer]] framework manages the final rendering to the platform-specific window that you actually see as a user. It maintains a list of `Source` elements that provide platform-specific rendering logic, with one such source for each active Scene in a GUI (e.g., a dialog Scene could be on top of a main window Scene).

There are special Sources for _direct rendering_ elements such as the [[xyz]] 3D Scene, or a [[video]] element, which render directly to the screen in an optimized, no-overhead manner.

For a `Scene`, the `SceneSource` manages getting the `render.Render` list from the `paint.Painter` on the `Scene`, and sends it to the platform-specific `render.Renderer` which is either the Go-based `rasterx` renderer or the web html canvas renderer, and then manages the display of the resulting image to the window.

For desktop and mobile, the final display is done using the [[gpu]] system that can "blit" the rendered `image.RGBA` image directly to the display window very efficiently using the GPU hardware. At some point, we will do the rendering on the GPU as well, in which case it will all happen directly on the GPU.

For web, the `Composer` manages a stack of `canvas` elements that are what you actually see on the browser, so the rendering updates to these canvas elements from the `render.Render` actions directly updates the visible display, without any further upload. Direct rendering of [[xyz]] happens by virtue of managing a webgpu canvas element that the xyz [[gpu]] system directly renders to.

## Structure of the renderWindow

The overall management of rendering is organized as follows:

* `renderWindow` uses the [[doc:system.Drawer]] interface (which is implemented using WebGPU on desktop, mobile, and web) to composite and "blit" (quickly render) all the individual image elements out to the actual hardware window that you see, layered in the proper order.  On the web and offscreen (testing) platforms, Drawer is implemented using Go `image.Draw` functions, with no other hardware dependencies.

* `stages` in the renderWindow manages a stack of `stage` elements, each of which manages an individual [[doc:core.Scene]].  There are different types of stage, specified by the [[doc:core.StageTypes]] enum, including the main `WindowStage` for primary app content, `DialogStage`, `MenuStage`, `TooltipStage` etc.

* The `stage` also manages a collection of [[doc:core.Sprites]] which provide lightweight graphical elements that are rendered over the top of the underlying content, for things like selection boxes, cursors, etc.

* The [[doc:core.Scene]] is the base-level element that contains a coherent set of Widget elements, which are subject to mutual layout constraints and are all rendered onto a shared image.  It specifies the _content_ of a GUI element, whereas the stage has parameters that control the overall behavior and event processing, which are different for popups vs main windows.

