package maputils

// Contains is a shortcut to `_, ok := map[key]`. This allows for evaluating
func Contains[T comparable, V any](m map[T]V, value T) bool {
	_, ok := m[value]
	return ok
}
