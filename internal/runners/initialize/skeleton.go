package initialize

import "strings"

// Skeleton constants represent available skeleton styles.
const (
	SkeletonBase   = "base"
	SkeletonEditor = "editor"
)

var styleLookup = []string{
	SkeletonBase,
	SkeletonEditor,
}

func styleRecognized(style string) bool {
	for _, token := range styleLookup {
		if token == style {
			return true
		}
	}
	return false
}

// RecognizedSkeletonStyles returns a CSV list of recognized skeleton style
// values.
func RecognizedSkeletonStyles() string {
	return strings.Join(styleLookup, ", ")
}
