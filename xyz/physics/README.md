# XYZ Physics Engine

This `physics` engine is a scenegraph-based physics simulator for creating virtual environments.  It provides a `Body` for 

Ultimately we hope to figure out how the [Bullet](https://github.com/bulletphysics/bullet3) system works and get that running here, in a clean and simple implementation.

Incrementally, we will start with a basic explicitly driven form of physics that is sufficient to get started, and build from there.

The [world](world) visualization is made using [GoKi](https://github.com/goki/ki) based trees (groups, bodies, joints).

Rendering can *optionally* be performed using corresponding 3D renders in the `xyz` 3D rendering framework in the [Cogent Core](https://cogentcore.org/core) GUI framework, using an `epev.View` object that sync's the two.

We also use the full-featured `math32` math / matrix library (adapted from the `g3n` 3D game environment package).

# Organizing the World

It is most efficient to create a relatively deep tree with `Group` nodes that collect nearby `Body` objects at multiple levels of spatial scale.  Bounding Boxes are computed at every level of Group, and pruning is done at every level, so large chunks of the tree can be eliminated easily with this strategy.

Also, Nodes must be specifically flagged as being `Dynamic` -- otherwise they are assumed to be static -- and each type should be organized into separate top-level Groups (there can be multiple of each, but don't mix Dynamic and Static).  Static nodes are never collided against each-other.  Ideally, all the Dynamic nodes are in separate top-level, or at least second-to-top level groups -- this eliminates redundant A vs. B and B vs. A collisions and focuses each collision on the most relevant information.

# Updating Modes 

There are two major modes of updating: Scripted or Physics -- scripted requires a program to control what happens on every time step, while physics uses computed forces from contacts, plus joint constraints, to update velocities (not yet supported).  The update modes are just about which methods you call.

The `Group` has a set of `World*` methods that should be used on the top-level world Group node node to do all the init and update steps. The update loops automatically exclude non Dynamic nodes.

* `WorldInit` -- everyone calls this at the start to set the initial config

* `WorldRelToAbs` -- for scripted mode when updating relative positions, rotations.

* `WorldStepPhys` -- for either scripted or physics modes, to update from current velocities.

* `WorldCollide` -- returns list of potential collision contacts based on projected motion, focusing on dynamic vs. static and dynamic vs. dynamic bodies, with optimized tree filtering.  This is the first pass for collision detection.  
 
## Scripted Mode

For Scripted mode, each update step typically involves manually updating the `Rel.Pos` and `.Quat` fields on `Body` objects to update their relative positions.  This field is a `Phys` type and has `MoveOnAxis` and `RotateOnAxis` (and a number of other rotation methods).  The Move methods update the `LinVel` field to reflect any delta in movement.

It is also possible to manually set the `Abs.LinVel` and `Abs.AngVel` fields and call `StepPhys` to update.

For collision detection, it is essential to have the `Abs.LinVel` field set to anticipate the effects of motion and determine likely future impacts.  The RelToAbs update call does this automatically, and if you're instead using StepPhys the LinVel is already set.  Both calls will automatically compute an updated BBox and VelBBox.

It is up to the user to manage the list of potential collisions, e.g., by setting velocity to 0 or bouncing back etc.

## Physics Mode

The good news so far is that the full physics version as in Bullet is actually not too bad.  The core update step is a super simple forward Euler, intuitive update (just add velocity to position, with a step size factor).  The remaining work is just in computing the forces to update those velocities.  Bullet uses a hybrid approach that is clearly described in the [Mirtich thesis](https://people.eecs.berkeley.edu/~jfc/mirtich/thesis/mirtichThesis.pdf), which combines *impulses* with a particular way of handling joints, due originally to Featherstone.  Impulses are really simple conceptually: when two objects collide, they bounce back off of each other in proportion to their `Bounce` (coefficient of restitution) factor -- these collision impact forces dominate everything else, and aren't that hard to compute (similar conceptually to the `marbles` example in GoGi).  The joint constraint stuff is a bit more complicated but not the worst.  Everything can be done incrementally.  And the resulting system will avoid the brittle nature of the full constraint-based approach taken in ODE, which caused a lot of crashes and instability in `cemer`.

One of the major problems with the impulse-based approach: that it causes otherwise "still" objects to jiggle around and slip down planes, seems eminently tractable with special-case code that doesn't seem too hard.

more info: https://caseymuratori.com/blog_0003

