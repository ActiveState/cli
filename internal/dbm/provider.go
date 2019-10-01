package dbm

type ProjectProvider interface {
	ProjectByOrgAndName(org, name string) (*ProjectResp, error)
}
