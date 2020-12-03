package sliceutils

import "golang.org/x/text/unicode/norm"

func RemoveFromStrings(slice []string, n int) []string {
	return append(slice[:n], slice[n+1:]...)
}

func GetInt(slice []int, index int) (int, bool) {
	if index > len(slice)-1 {
		return -1, false
	}
	return slice[index], true
}

func GetString(slice []string, index int) (string, bool) {
	if index > len(slice)-1 {
		return "", false
	}
	// return normalized string
	return norm.NFC.String(slice[index]), true
}

func IntRangeUncapped(in []int, start, end int) []int {
	if end > len(in) {
		end = len(in)
	}
	return in[start:end]
}