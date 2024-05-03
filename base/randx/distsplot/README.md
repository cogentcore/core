# distplot

This executable plots a histogram of random numbers generated according to different distributions, according to the `RndParams` parameterized distributions.

Here are the distributions and how the parameters in `RndParams` map onto distributional parameters -- the `Mean` and `Var` are not the actual mean and variance of the distribution, but rather provide parameters roughly corresponding to these values, along with the extra `Par` value:

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

The range of these distributions vary so you'll have to adjust the Range values as you try different distributions and parameters.

