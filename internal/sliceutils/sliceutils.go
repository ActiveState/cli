package sliceutils

func RemoveFromStrings(slice []string, n int) []string {
	return append(slice[:n], slice[n+1:]...)
}
