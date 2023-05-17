package parser

import (
	"fmt"
	"regexp"
	"strings"
)

// CleanInput expects a string that is a JSON object with quoted keys and
// values. It returns the same JSON object with unquoted keys.
func CleanInput(input string) string {
	re := regexp.MustCompile(`"(\w+)":`)
	matches := re.FindAllString(input, -1)

	for _, match := range matches {
		unquotedKey := strings.Trim(match, `":`)
		input = strings.ReplaceAll(input, match, fmt.Sprintf("%s:", unquotedKey))
	}

	return input
}
