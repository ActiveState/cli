package request

type HashGlobs struct {
	wd    string
	globs []string
}

func NewHashGlobs(wd string, globs []string) *HashGlobs {
	return &HashGlobs{wd: wd, globs: globs}
}

func (c *HashGlobs) Query() string {
	return `query(wd: String!, $globs: [String!]!) {
	hashGlobs(wd: $wd, globs: $globs)  {
		hash
		files {
			pattern
			path
			hash
		}
	}
}`
}

func (c *HashGlobs) Vars() (map[string]interface{}, error) {
	return map[string]interface{}{
		"wd":    c.wd,
		"globs": c.globs,
	}, nil
}
