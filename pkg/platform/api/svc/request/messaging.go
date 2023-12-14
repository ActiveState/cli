package request

type MessagingRequest struct {
	command string
	flags   []string
}

func NewMessagingRequest(command string, flags []string) *MessagingRequest {
	return &MessagingRequest{
		command: command,
		flags:   flags,
	}
}

func (m *MessagingRequest) Query() string {
	return `query($command: String!, $flags: [String!]!)  {
		checkMessages(command: $command, flags: $flags) {
			id
			message
			interrupt
			placement
		}
	}`
}

func (m *MessagingRequest) Vars() map[string]interface{} {
	return map[string]interface{}{
		"command": m.command,
		"flags":   m.flags,
	}
}
