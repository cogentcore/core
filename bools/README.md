# bools

package bools does conversion to / from booleans and other go standard types:

```Go
// ToFloat32 converts a bool to a 1 (true) or 0 (false)
func ToFloat32(b bool) float32 {
	if b {
		return 1
	}
	return 0
}

// FromFloat32 converts value to a bool, 0 = false, else true
func FromFloat32(v float32) bool {
	if v == 0 {
		return false
	}
	return true
}
```

Other types are: `float64, int, int64, int32` -- you can cast from there as necessary.

