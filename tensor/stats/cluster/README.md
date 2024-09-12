# cluster

`cluster` implements agglomerative clustering of items based on [metric](../metric) distance `Matrix` data (which is provided as an input, and must have been generated with a distance-like metric (increasing with dissimiliarity).

There are different standard ways of accumulating the aggregate distance of a node based on its leaves:

* `Min`: the minimum-distance across leaves, i.e., the single-linkage weighting function.
* `Max`: the maximum-distance across leaves, i.e,. the complete-linkage weighting function.
* `Avg`: the average-distance across leaves, i.e., the  average-linkage weighting function.
* `Contrast`:  is Max + (average within distance - average between distance).

`GlomCluster` is the main function, taking different `ClusterFunc` options for comparing distance between items.


