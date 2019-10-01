package rxutils

import (
	"regexp"
)

// ReplaceAllStringSubmatchFunc allows you to replace regex with a callback that includes matched groups
// based on https://medium.com/@elliotchance/go-replace-string-with-regular-expression-callback-f89948bad0bb
func ReplaceAllStringSubmatchFunc(re *regexp.Regexp, input string, replacer func([]string) string) string {
	result := ""
	lastIndex := 0
	for _, v := range re.FindAllSubmatchIndex([]byte(input), -1) {
		groups := []string{}
		for i := 0; i < len(v); i += 2 {
			if v[i] == -1 {
				continue // optional group
			}
			groups = append(groups, input[v[i]:v[i+1]])
		}
		result += input[lastIndex:v[0]] + replacer(groups)
		lastIndex = v[1]
	}
	return result + input[lastIndex:]
}
