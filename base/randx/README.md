# randx

Package randx provides randomization functionality built on top of standard `math/rand`
random number generation functions.  Includes:
*  RandParams: specifies parameters for random number generation according to various distributions used e.g., for initializing random weights and generating random noise in neurons
*  Permute*: basic convenience methods calling rand.Shuffle on e.g., []int slice

Here are the distributions and how the parameters in `RandParams` map onto distributional parameters -- the `Mean` and `Var` are not the actual mean and variance of the distribution, but rather provide parameters roughly corresponding to these values, along with the extra `Par` value:

```Go
	// Binomial represents number of 1's in n (Par) random (Bernouli) trials of probability p (Var)
	Binomial

	// Poisson represents number of events in interval, with event rate (lambda = Var) plus Mean
	Poisson

	// Gamma represents maximum entropy distribution with two parameters: scaling parameter (Var)
	// and shape parameter k (Par) plus Mean
	Gamma

	// Gaussian normal with Var = stddev plus Mean
	Gaussian

	// Beta with Var = alpha and Par = beta shape parameters
	Beta

	// Mean is just the constant Mean, no randomness
	Mean
```

See [distplot](distplot) for a program to plot the histograms of these different distributions as you vary the parameters.

