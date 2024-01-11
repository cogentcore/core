This example tests for numerical differences between the GPU and CPU, running the same code on the same numbers on both sides and comparing the results.

In axon, we have divergences from GABA and NMDA conductances, and these appear to be inevitable, given the outcome of these tests.

Here's a summary of the results:

```
    // Out[idx.x] = FastExp(In[idx.x]); // 0 diffs
	float vbio = In[idx.x];
	float eval = 0.1 * ((vbio + 90.0) + 10.0);
	// Out[idx.x] = (vbio + 90.0) / (1.0 + FastExp(eval)); // lots of diffs
	// Out[idx.x] = (vbio + 90.0) / (1.0 + exp(eval)); // worse
	// Out[idx.x] = eval; // 0 diff
	// Out[idx.x] = 1.0 / eval; // a few 2.98e-8 diffs already!
	Out[idx.x] = 1.0 / FastExp(eval); // lots more diffs, e-08, -09
```

Interestingly, FastExp itself does not have any diffs but even simple 1.0 / division introduces issues.

