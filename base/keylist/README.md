# keylist

keylist implements an ordered list (slice) of items (Values), with a map from a Key (e.g., names) to indexes, to support fast lookup by name.  There is also a Keys slice.

This is a different implementation of the [ordmap](../ordmap) package, and has the advantage of direct slice access to the values, instead of having to go through the KeyValue tuple struct in ordmap.

