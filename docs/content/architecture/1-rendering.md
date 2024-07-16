## Rendering logic

The logic for rendering updates to the 

Key principles:

* Async updates (animation, mouse events, etc) change state, _set only flags_
  using thread-safe atomic bitflag operations. True async rendering
  is really hard to get right, and requires tons of mutexes etc. Async updates
 must go through [WidgetBase.AsyncLock] and [WidgetBase.AsyncUnlock].
* Synchronous, full-tree render updates do the layout, rendering,
  at regular FPS (frames-per-second) rate -- nop unless flag set.

Three main steps:
* Config: (re)configures widgets based on current params
  typically by making Parts.  Always calls ApplyStyle.
* Layout: does sizing and positioning on tree, arranging widgets.
  Needed for whole tree after any Config changes anywhere
  See layout.go for full details and code.
* Render: just draws with current config, layout.

ApplyStyle is always called after Config, and after any
current state of the Widget changes via events, animations, etc
(e.g., a Hover started or a Button is pushed down).
Use NeedsRender() to drive the rendering update at next DoNeedsRender call.

The initial configuration of a scene can skip calling
Config and ApplyStyle because these will be called automatically
during the Run() process for the Scene.

For dynamic reconfiguration after initial display,
Update() is the key method, calling Config then
ApplyStyle on the node and all of its children.

For nodes with dynamic content that doesn't require styling or config,
a simple NeedsRender call will drive re-rendering.

Updating is _always_ driven top-down by RenderWindow at FPS sampling rate,
in the DoUpdate() call on the Scene.
Three types of updates can be triggered, in order of least impact
and highest frequency first:
* ScNeedsRender: does NeedsRender on nodes.
* ScNeedsLayout: does GetSize, DoLayout, then Render -- after Config.

