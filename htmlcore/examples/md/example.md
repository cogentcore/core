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

## more lists
	
The relevant background for this algorithm is presented in the following pages: 

* Synaptic plasticity reviews the relevant neuroscience for how synapses change their effective strength (weight), and the critical contributions of kinases to this process.

* Temporal derivative provides a high-level account for the essential computational principles behind this algorithm, including an interactive simulation of how a competitive interaction between fast and slow pathways can compute the _error gradient_ at the heart of error-driven learning.

* GeneRec derives a concrete learning algorithm directly from the mathematics of error backpropagation, which uses bidirectional connectivity to propagate error gradients throughout the neocortex. The kinase algorithm is closely related to GeneRec.

Here, we build on these foundations to describe the detailed mechanisms that actually drive learning in the Axon models, which represent an attempt to satisfy constraints from neuroscience, computational efficacy, and computational cost.

At a big-picture level, the two central ideas behind the kinase algorithm are:

* Use biophysically-grounded equations for computing the synaptic Ca++ influx via the neuron channels#NMDA and neuron channels#VGCC channels that are well-established as the primary initial drivers of synaptic plasticity. NMDA is sensitive to the conjunction of pre and postsynaptic activity, while VGCCs (voltage-gated calcium channels) are driven in a sharply phasic manner by backpropagating action potentials from the receiving neuron.

* Apply a cascade of simple exponential integration steps to simulate the complex biochemical processes that follow from this Ca++ influx, with time constants optimized based on computational performance across a wide range of tasks. The final two steps in this cascade implement the temporal derivative computation where the faster penultimate step drives LTP (weight increases) while the final slower step drives LTD (weight decreases).

	This strategy leverages biophysically constrained mechanisms where they are well-established, while adopting a more abstracted computationally-motivated approach to the complexities of the subsequent biochemical processes, which are not yet sufficiently specified to support a more bottom-up approach. The overall mechanism behind the temporal derivative is supported by the general properties of the CaMKII and DAPK1 kinases and related mechanisms, as described in synaptic plasticity, and by the initial empirical results of Jiang et al 2025.
	
	Furthermore, other things are good about this.

    $$
    a = b * c
    $$
    
However, at a pragmatic implementational level, it would be very expensive to compute the Ca++ influx based on the NMDA and VGCC biophysical equations for each synapse individually, given that synapses greatly outnumber neurons (e.g., $N^2$ in a fully-connected model), Therefore, we instead break out the computation into two subcomponents:

* A shared dendrite-level Ca++ value that reflects the overall dendritic membrane potential and the contributions of backpropagating action potentials from the receiving neuron on the NMDA and VGCC channels.

* An efficiently-computed synapse-specific multiplier, that reflects the specific coincident pre and postsynaptic activity at each synapse.

In another example of the synergies between neuroscience and computation, these two terms can be directly associated with the two essential terms in the error backpropagation learning rule, which are the _error gradient_ and the _credit assignment_ factors:
	
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

{style="height:10em"}
![Image of the Go Gopher](https://miro.medium.com/v2/resize:fit:1000/0*YISbBYJg5hkJGcQd.png)

<h3 style="color:red">This is some HTML:</h3>

<button>Click me!</button>

## Divs

Here is some text

{style="min-height:5em;max-height:10em"}
<div>

This _text_ is in a `div`. it should be **fine**, just in a separate frame.

</div>

This is the text after the div.

## Tables

| Channel Type     | Tau (ms) |
|------------------|----------|
| Fast (M-type)    | 50       |
| Medium (Slick)   | 200      |
| Slow (Slack)     | 1000     |
|      | 500 |
| more | 2000 |
| and  | 100 |

Here is text after the table

