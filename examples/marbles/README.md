# GoGi Marbles Example

This is a fun interactive app, based on the marble version of [desmos](https://www.desmos.com/).  Originally written by Kai O'Reilly with some help from his dad :)  See [kplat1](https://github.com/kplat1) for his github page.

## Installation

This example has an additional dependency beyond those of GoGi: 	[govaluate](https://github.com/Knetic/govaluate) -- easiest to use the following procedure to get it:

The usual Go install procedure will work -- this is the easiest way to install GoGi and get all of its various dependencies:

``` bash
> go get github.com/goki/gi
> cd ~/go/src/github.com/goki/gi/examples/marbles
> go get ...
> go build
> ./marbles
```

## Tips

Start by loading some of the existing setups, using the Open button.  Just hit Run to see the marbles go.

Play with the parameters at the top -- they take effect immediately.  The bounce level for each line can be set separately.  Bounce = 1 means it is perfectly elastic, and watch out if you set it greater than 1!

The graph area is -10..+10 in both axes.

Pay attention to the tooltip for the equation field for special constraints.

