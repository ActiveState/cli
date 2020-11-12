package sliceutils

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
	return slice[index], true
}
