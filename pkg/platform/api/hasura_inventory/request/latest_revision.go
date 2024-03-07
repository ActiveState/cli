package request

type LatestRevision struct {
}

func NewLatestRevision() *LatestRevision {
	return &LatestRevision{}
}

func (l *LatestRevision) Query() string {
	return `query last_ingredient_revision_time {
		last_ingredient_revision_time {
			revision_time
		}
	}`
}

func (l *LatestRevision) Vars() (map[string]interface{}, error) {
	return nil, nil
}
