## GLFW backend for go.wde

### Notes:

- glfw doesn't support icons for the windows (except on the MS Windows platform).
  To implement this, platform specific code needs to be written. However, the 
  issue is known at glfw, so in the future it will be probably implemented.
- glfw supports clipboard messaging. This is not implemented (yet) in go.wde 
  but IMO it should be part of the event system.
- glfw key numpad works with numbers and go.wde with arrows/ins/del/pageup/etc.
  So numpad key 5 has no go.wde mapping!
- go.wde has a key called KeyFunction and I don't know what it is. There is no
  direct glfw counterpart for this AFAIK.
- I have no idea how to get the status of capslock, numlock, pause, scrollock.
  This means that when the user presses the letter 'a', it is unclear that the 
  output is 'a' or 'A'. This of course US English only.

### Open issues:

- Still not all events are handled
- Add Icons to the windows.
- Window.FlushImage(). How to implement "bounds ...image.Rectangle" ?
- Sometimes when exiting, the following error (numbers differ) is displayed:

<b></b>

    X Error of failed request:  BadWindow (invalid Window parameter)
      Major opcode of failed request:  3 (X_GetWindowAttributes)
      Resource id in failed request:  0x0
      Serial number of failed request:  963
      Current serial number in output stream:  964

- Only tested on Ubuntu 12.04 64-bit
- I am not sure that glfw.PollEvents() is implemented correctly in combination
  with multiple windows. Also glfw.SwapInterval() isn't implemented (yet).

### Solved issues:

- Where to place glfw3.Init()? go.wde doesn't work with Init(), 
  but glfw3.Init() must be run in the main thread. So placing it in the
  func wdeglfw3.init() might not solve it.
  I will test it in the gears app.
  Answ: Solved.
- How to access OpenGL functions? This is not a real issue for go.wde because
  only At() is being used. But for the other draw2d functions in go.uik it 
  needs to be fully implemented.
  Answ: glfw3 is only used as a backend for go.wde. That means the OpenGL
  functionality is not available in go.uik.
- How to access the buffers of a window?
  Answ: Don't. Let OpenGL handle the rendering of a window.
- How to implement glfw3.SwapBuffers? go.wde solves this (I guess) with
  CopyRGBA(), but glfw3 doesn't.
  Answ: By calling FlushImage() and let this function call SwapBuffers.
- How to properly shutdown the app? Why is this a problem? 
  Because glfw3.Main() is blocking a proper shutdown.
  Answ:

<b></b>

        wde.BackendStop = func() {
            glfw.Terminate()
            // other cleanup functions
            os.Exit(0)
        }

- The individual window rendering is done by calling 
  glfw.MakeContextCurrent. That means only one window is rendered at a time
  and kinda ruins the goroutine idea.
  Let's see how to implement it without modifying wdetest.
  Answ: Looks like that implementing it in Screen() is the easiest.
  The blocking is fixed with channels.
- The event system of glfw3 works very different than that of go.wde
  However the callback functions might/should solve that
  Answ1: Added Window.C() in package glfw3 to get access of the underlying C
  structure pointer.
- LockSize() in struct wdeglfw3.Window modifies Window.lockedSize.
  What is the functionality of this function?
  Answ: It a switch for window stretching allowment (think dialogs).
  However, in glfw this switch need to be set before the window initialization.
- Drawing the window buffer on the screen is a serious issue. In OpenGL there 
  are two ways to do this:
    - gl.DrawPixels (deprecated in OpenGL 4.0). Problem is vertical window 
      stretching. Answ: For now, the stretching issue is solved with 
      gl.Viewport. So we stick with gl.DrawPixels. Maybe in the future it is 
      better to switch to gl.TexImage2D.
    - gl.TexImage2D. Problem is that it only shows a white square on the 
      first quadrant (right upper corner)
  
