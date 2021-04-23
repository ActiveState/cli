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
	return `query($c: String, $v: String) {
		update(channel: $c, version: $v) {
			channel
			version
			logfile
		}
	}`
}

func (u *UpdateRequest) Vars() map[string]interface{} {
	return map[string]interface{}{
		"c": u.channel,
		"v": u.version,
	}
}
