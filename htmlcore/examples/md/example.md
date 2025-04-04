# htmlcore in Markdown

### Made with ***Cogent Core***

This is a sample _MD_ (markdown) **document** displayed using `htmlcore`.

This is a [link to the ***Cogent Core*** website](https://cogentcore.org/core), which you can _click_ on to see helpful **documentation** and examples for the *Cogent Core* framework.

You can include math: $ a = f(x^2) $ inline and:

$$
y = \frac{1}{N} \left( \sum_{i=0}^{100} \frac{f(x^2)}{\sum x^2} \right)
$$

as a standalone item.

# Heading 1
## Heading 2
### Heading 3
#### Heading 4
##### Heading 5
###### Heading 6

* List item 1
    * Sub list item 1
    * Sub list item 2
* List item 2
    * Sub list item 1
    * Sub list item 2
* List item 3
    * Sub list item 1
    * Sub list item 2

1. List item 1
    1. Sub list item 1
    2. Sub list item 2
2. List item 2
    1. Sub list item 1
    2. Sub list item 2
3. List item 3
    1. Sub list item 1
    2. Sub list item 2


* List item that has indented item below it.

	Indented list item that should follow the indentation from above but not have a *

> quote element
	
	
### This is a code block:

```go
package main

import "fmt"

func main() {
    fmt.Println("Hello, world!")
}
```

### This is a collapsed code block:

{collapsed="true"}
```go
package main

import "fmt"

func main() {
    fmt.Println("Hello, world!")
}
```

### This is an image of the Go Gopher: 

![Image of the Go Gopher](https://miro.medium.com/v2/resize:fit:1000/0*YISbBYJg5hkJGcQd.png)

<h3 style="color:red">This is some HTML:</h3>

<button>Click me!</button>

## Divs

Here is some text

<div>

This _text_ is in a `div`. it should be **fine**, just in a separate frame.

</div>

This is the text after the div.

