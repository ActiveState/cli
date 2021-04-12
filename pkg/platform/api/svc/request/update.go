package request

type UpdateRequest struct {
	channel string
	version string
}

func NewUpdateRequest(channel, version string) *UpdateRequest {
	return &UpdateRequest{
		channel: channel,
		version: version,
	}
}

func (u *UpdateRequest) Query() string {
	return `query {
		update(channel: $channel, version: $version) {
			channel
			version
		}
	}`
}

func (u *UpdateRequest) Vars() map[string]interface{} {
	return map[string]interface{}{
		"channel": u.channel,
		"version": u.version,
	}
}
