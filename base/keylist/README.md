# keylist

keylist implements an ordered list (slice) of items (Values), with a map from a Key (e.g., names) to indexes, to support fast lookup by name.  There is also a Keys slice.

It has separate slices for the values and keys, for direct access to them, and `All` iterates over both.

