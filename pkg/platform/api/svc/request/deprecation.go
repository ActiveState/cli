package request

type DeprecationRequest struct{}

func NewDeprecationRequest() *DeprecationRequest {
	return &DeprecationRequest{}
}

func (d *DeprecationRequest) Query() string {
	return `query {
		checkDeprecation {
			deprecated
			version
			date
			dateReached
			reason
		}
	}`
}

func (d *DeprecationRequest) Vars() map[string]interface{} {
	return nil
}
