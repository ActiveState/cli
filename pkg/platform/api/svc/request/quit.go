package request

type QuitRequest struct{}

func NewQuitRequest() *QuitRequest {
	return &QuitRequest{}
}

func (q *QuitRequest) Query() string {
	return `subscription {
		quit
	}`
}

func (q *QuitRequest) Vars() map[string]interface{} {
	return nil
}
