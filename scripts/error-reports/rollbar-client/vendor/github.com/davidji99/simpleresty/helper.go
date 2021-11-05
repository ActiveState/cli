package simpleresty

import "strings"

func contains(a []string, x string, ignoreCase bool) bool {
	for _, n := range a {
		if ignoreCase {
			if strings.ToLower(x) == strings.ToLower(n) {
				return true
			}
		} else {
			if x == n {
				return true
			}
		}
	}
	return false
}
