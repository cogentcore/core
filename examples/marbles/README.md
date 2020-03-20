# GoGi Marbles Example

This is a fun interactive app, based on the marble version of [desmos](https://www.desmos.com/).  Originally written by [kplat1](https://github.com/kplat1) -- see: [kplat1/marbles](https://github.com/kplat1/marbles).

## Installation

This example has an additional dependency beyond those of GoGi: [govaluate](https://github.com/Knetic/govaluate) -- easiest to use the following procedure to get it:

The usual Go install procedure will work -- this is the easiest way to install GoGi and get all of its various dependencies:

``` bash
$ go get github.com/goki/gi
$ cd ~/go/src/github.com/goki/gi/examples/marbles
$ go get ./...
$ go build
$ ./marbles
```

## Tips

Start by loading some of the existing setups, using the Open button.  Just hit Run to see the marbles go.

Play with the parameters at the top -- they take effect immediately.  The bounce level for each line can be set separately.  Bounce = 1 means it is perfectly elastic, and watch out if you set it greater than 1!

The graph area is -10..+10 in both axes. 

Pay attention to the tooltip for the equation field for special constraints -- must start numbers with a 0 for example.

Extra points to anyone who can figure out what is causing the balls to occasionally quantum tunnel their way out of things!  Not sure we want to fix it, but would be good to at least know why it is happening :)

 *News:* we finally figured out how to separate the marbles, using a random X initial starting component -- see the     `Width` parameter -- works great.

## Implementational notes

All drawing is done using SVG nodes -- look in `graph.go` for details, e.g., `UpdateMarbles` is the main update for the marbles -- just wraps the SvgGraph in `UpdateStart` and `UpdateEnd` calls and updates the positions of the marble `svg.Circle` nodes.

The app takes good advantage of the `ValueView` elements so the GUI code in `main.go` is really minimal -- everything is handled by the `giv.StructView` and `giv.TableView` view elements.  The main toolbar for the app generated from the type properties on `Graph` object: `GraphProps` in `graph.go`.  See  [Views](https://github.com/goki/gi/wiki/Views) for more details.


