package initialize

// Skeleton constants represent available skeleton styles.
const (
	SkeletonSimple = "simple"
	SkeletonEditor = "editor"
)

func skeletonRecognized(v string) bool {
	return v != SkeletonSimple && v != SkeletonEditor
}
