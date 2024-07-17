## Rendering logic

At the highest level, rendering is made robust by having a completely separate, mutex lock-protected pass where all render-level updating takes place.  This render pass is triggered by [[events.WindowPaint]] events that are sent regularly at 60 FPS (frames per second).  If nothing needs to be updated, nothing happens (which is the typical case for most frames), so it is not a significant additional cost.

The usual processing of events that arise in response to user GUI actions, or any other source of changes, sets flags that determine what kind of updating needs to happen during rendering.  These are typically set via [[WidgetBase.NeedsRender]] or [[WidgetBase.NeedsLayout]] calls.

The first step in the `renderWindow.renderWindow()` rendering function is to call `updateAll()` which ends up calling `doUpdate()` on the [[Scene]] elements within a render window, and this function is what checks if a new [layout](layout) pass has been called for, or whether any individual widgets need to be rendered. This rendering update writes to a separate `image.RGBA` owned by the Scene, which provides the raw input for the final image rendered to the window.

Most updating of widgets happens in the event processing step, which is synchronous (one event is processed at a time).  

For any updating that happens outside of the normal event loop (e.g., timer-based animations etc), you must go through [[WidgetBase.AsyncLock]] and [[WidgetBase.AsyncUnlock]].  The [[WidgetBase.Async]] helper function makes this a one-liner.

## Structure of the renderWindow

* stages, etc

ApplyStyle is always called after Config, and after any current state of the Widget changes via events, animations, etc (e.g., a Hover started or a Button is pushed down). Use NeedsRender() to drive the rendering update at next DoNeedsRender call.

The initial configuration of a scene can skip calling Config and ApplyStyle because these will be called automatically during the Run() process for the Scene.

For dynamic reconfiguration after initial display, Update() is the key method, calling Config then ApplyStyle on the node and all of its children.

For nodes with dynamic content that doesn't require styling or config, a simple NeedsRender call will drive re-rendering.

Updating is _always_ driven top-down by RenderWindow at FPS sampling rate, in the DoUpdate() call on the Scene. Three types of updates can be triggered, in order of least impact and highest frequency first:
* ScNeedsRender: does NeedsRender on nodes.
* ScNeedsLayout: does GetSize, DoLayout, then Render -- after Config.

