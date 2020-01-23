package preprocess

type versionService struct {
	github *githubClient
	branch string
}

func newVersionService(github *githubClient, branchName string) *versionService {
	return &versionService{
		github: github,
		branch: branchName,
	}
}

func (s *versionService) currentVersion() (string, error) {
	return "TODO:", nil
}

func (s *versionService) getVersion() (string, error) {
	return "TODO:", nil
}

func (s *versionService) getVersionPreRelease() (string, error) {
	return "TODO:", nil
}
