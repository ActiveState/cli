package request

type NotificationRequest struct {
	command string
	flags   []string
}

func NewNotificationRequest(command string, flags []string) *NotificationRequest {
	return &NotificationRequest{
		command: command,
		flags:   flags,
	}
}

func (m *NotificationRequest) Query() string {
	return `query($command: String!, $flags: [String!]!) {
		checkNotifications(command: $command, flags: $flags) {
			id
			notification
			interrupt
			placement
		}
	}`
}

func (m *NotificationRequest) Vars() (map[string]interface{}, error) {
	return map[string]interface{}{
		"command": m.command,
		"flags":   m.flags,
	}, nil
}
