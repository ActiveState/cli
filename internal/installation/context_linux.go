package installation

type Context struct{}

func GetContext() (*Context, error) {
	return nil, nil
}

func SaveContext(context *Context) error {
	return nil
}

func getAdminInstallInformation() (bool, error) {
	return false, nil
}
