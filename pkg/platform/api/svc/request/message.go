package request

type MessageRequest struct {
}

func NewMessageRequest() *MessageRequest {
	return &MessageRequest{}
}

func (m *MessageRequest) Query() string {
	return `query {
		checkMessages {
			topic
			message
		}
	}`
}

func (m *MessageRequest) Vars() (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}
