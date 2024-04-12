// Package bitflags contains helper functions for working with bitmask enums.
package bitflags

// integer is primitive signed and unsigned integer types.
type integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | 
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

// Set flag in b to active state. More than 1 flag can be set active in a single 
// call by specifying other flags.
func Set[T integer](b, flag T, other ...T) T {
	if len(other) == 0 {
		return b|flag
	}
	c := b|flag
	for _, v := range other {
		c = c|v
	}
	return c
}

// Del sets flag in b to inactive state. More than 1 flag can be set inactive in 
// a single call by specifying other flags.
func Del[T integer](b, flag T, other ...T) T {
	if len(other) == 0 {
		return b&^flag
	}
	c := b&^flag
	for _, v := range other {
		c = c&^v
	}
	return c
}

// Toggle flips the state of flag in b. More than 1 flag can be toggled in a 
// single call by specifying other flags.
func Toggle[T integer](b, flag T, other ...T) T {
	if len(other) == 0 {
		return b^flag
	}
	c := b^flag
	for _, v := range other {
		c = c^v
	}
	return c
}

// Has returns true if flag and all other flags in b are in the active state.
func Has[T integer](b, flag T, other ...T) bool {
	if len(other) == 0 {
		return b&flag != 0
	}
	if b&flag == 0 {
		return false
	}
	for _, v := range other {
		if b&v == 0 {
			return false
		}
	}
	return true
}

// HasAny returns true if flag or any other flags in b is in the active state.
func HasAny[T integer](b, flag T, other ...T) bool {
	if len(other) == 0 {
		return b&flag != 0
	}
	if b&flag != 0 {
		return true
	}
	for _, v := range other {
		if b&v != 0 {
			return true
		}
	}
	return false
}
