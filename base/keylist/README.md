# keylist

keylist implements an ordered list (slice) of items, with a map from a key (e.g., names) to indexes, to support fast lookup by name.

Compared to the [ordmap](../ordmap) package, this is not as efficient for operations such as deletion and insertion, but it has the advantage of providing a simple slice of the target items that can be used directly in many cases.

Thus, it is more suitable for largely static lists, which are constructed by adding items to the end of the list.


