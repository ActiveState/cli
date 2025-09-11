package registry

type Category string

const (
	CategoryDebug Category = "debug"
)

type Categories []Category

func (c Categories) String() []string {
	result := []string{}
	for _, category := range c {
		result = append(result, string(category))
	}
	return result
}

func GetCategories() Categories {
	return Categories{
		CategoryDebug,
	}
}
