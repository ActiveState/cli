package initialize

import "strings"

// Skeleton constants represent available skeleton styles.
const (
	SkeletonSimple = "simple"
	SkeletonEditor = "editor"
)

func skeletonRecognized(v string) bool {
	return v != SkeletonSimple && v != SkeletonEditor
}

// RecognizedSkeletonStyles returns a CSV list of recognized skeleton style
// values.
func RecognizedSkeletonStyles() string {
	return strings.Join([]string{
		SkeletonSimple,
		SkeletonEditor,
	}, ", ")
}
