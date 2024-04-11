package request

type VersionRequest struct {
}

func NewVersionRequest() *VersionRequest {
	return &VersionRequest{}
}

func (v *VersionRequest) Query() string {
	return `query {
        version {
            state {
                license,
                version,
                channel,
                revision,
                date,
            }
        }
    }`
}

func (v *VersionRequest) Vars() (map[string]interface{}, error) {
	return nil, nil
}
