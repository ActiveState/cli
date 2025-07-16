package toolregistry

type ToolCategory string

const (
	ToolCategoryDebug ToolCategory = "debug"
)

type ToolCategories []ToolCategory

func (c ToolCategories) String() []string {
	result := []string{}
	for _, category := range c {
		result = append(result, string(category))
	}
	return result
}

func Categories() ToolCategories {
	return ToolCategories{
		ToolCategoryDebug,
	}
}