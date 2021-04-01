package request

type VersionRequest struct{}

func NewVersionRequest() *VersionRequest {
	return &VersionRequest{}
}

func (v *VersionRequest) Query() string {
	return `query {
        version {
            state {
                license,
                version,
                branch,
                revision,
                date,
            }
        }
    }`
}

func (v *VersionRequest) Vars() map[string]interface{} {
	return nil
}
