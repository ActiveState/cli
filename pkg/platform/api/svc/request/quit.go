package request

type QuitRequest struct{}

func NewQuitRequest() *QuitRequest {
	return &QuitRequest{}
}

func (v *QuitRequest) Query() string {
	return `query { quit { received } }`
}

func (v *QuitRequest) Vars() map[string]interface{} {
	return nil
}
