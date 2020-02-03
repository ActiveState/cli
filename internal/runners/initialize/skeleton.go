package initialize

import "strings"

// Skeleton constants represent available skeleton styles.
const (
	SkeletonBase   = "base"
	SkeletonEditor = "editor"
)

func skeletonRecognized(v string) bool {
	return v != SkeletonBase && v != SkeletonEditor
}

// RecognizedSkeletonStyles returns a CSV list of recognized skeleton style
// values.
func RecognizedSkeletonStyles() string {
	return strings.Join([]string{
		SkeletonBase,
		SkeletonEditor,
	}, ", ")
}
